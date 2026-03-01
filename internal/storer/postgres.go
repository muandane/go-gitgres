package storer

import (
	"context"
	"io"

	"go-gitgres/internal/db"

	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/format/index"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/go-git/go-git/v5/storage"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresStorer implements storage.Storer by storing objects and refs in PostgreSQL.
type PostgresStorer struct {
	pool   *pgxpool.Pool
	q      *db.Queries
	repoID int32
	ctx    context.Context
}

// NewPostgresStorer returns a storer for the given repo (by name). Repo is created if missing.
func NewPostgresStorer(ctx context.Context, pool *pgxpool.Pool, repoName string) (*PostgresStorer, error) {
	q := db.New(pool)
	repoID, err := q.GetOrCreateRepo(ctx, repoName)
	if err != nil {
		return nil, err
	}
	return &PostgresStorer{pool: pool, q: q, repoID: repoID, ctx: ctx}, nil
}

// NewPostgresStorerWithID returns a storer for an existing repo by ID (e.g. after GetRepo).
func NewPostgresStorerWithID(ctx context.Context, pool *pgxpool.Pool, repoID int32) *PostgresStorer {
	return &PostgresStorer{pool: pool, q: db.New(pool), repoID: repoID, ctx: ctx}
}

// RepoID returns the repository ID.
func (s *PostgresStorer) RepoID() int32 { return s.repoID }

var _ storage.Storer = (*PostgresStorer)(nil)

// EncodedObjectStorer
func (s *PostgresStorer) NewEncodedObject() plumbing.EncodedObject {
	return &plumbing.MemoryObject{}
}

func (s *PostgresStorer) SetEncodedObject(obj plumbing.EncodedObject) (plumbing.Hash, error) {
	r, err := obj.Reader()
	if err != nil {
		return plumbing.ZeroHash, err
	}
	defer r.Close()
	content, err := io.ReadAll(r)
	if err != nil {
		return plumbing.ZeroHash, err
	}
	oidRes, err := s.q.ObjectWrite(s.ctx, db.ObjectWriteParams{
		RepoID:  s.repoID,
		ObjType: int16(obj.Type()),
		Content: content,
	})
	if err != nil {
		return plumbing.ZeroHash, err
	}
	oid, ok := oidRes.([]byte)
	if !ok || len(oid) != 20 {
		return plumbing.ZeroHash, plumbing.ErrObjectNotFound
	}
	var h plumbing.Hash
	copy(h[:], oid)
	return h, nil
}

func (s *PostgresStorer) EncodedObject(t plumbing.ObjectType, h plumbing.Hash) (plumbing.EncodedObject, error) {
	row, err := s.q.ObjectRead(s.ctx, db.ObjectReadParams{RepoID: s.repoID, Oid: h[:]})
	if err != nil {
		return nil, plumbing.ErrObjectNotFound
	}
	if t != plumbing.AnyObject && plumbing.ObjectType(row.Type) != t {
		return nil, plumbing.ErrObjectNotFound
	}
	obj := &plumbing.MemoryObject{}
	obj.SetType(plumbing.ObjectType(row.Type))
	obj.SetSize(int64(row.Size))
	if _, err := obj.Write(row.Content); err != nil {
		return nil, err
	}
	return obj, nil
}

func (s *PostgresStorer) IterEncodedObjects(t plumbing.ObjectType) (storer.EncodedObjectIter, error) {
	oids, err := s.q.ListObjectOidsPrealloc(s.ctx, s.repoID)
	if err != nil {
		return nil, err
	}
	hashes := make([]plumbing.Hash, 0, len(oids))
	for _, oid := range oids {
		if len(oid) != 20 {
			continue
		}
		var h plumbing.Hash
		copy(h[:], oid)
		hashes = append(hashes, h)
	}
	return storer.NewEncodedObjectLookupIter(s, t, hashes), nil
}

func (s *PostgresStorer) HasEncodedObject(h plumbing.Hash) error {
	_, err := s.q.ObjectRead(s.ctx, db.ObjectReadParams{RepoID: s.repoID, Oid: h[:]})
	if err != nil {
		return plumbing.ErrObjectNotFound
	}
	return nil
}

func (s *PostgresStorer) EncodedObjectSize(h plumbing.Hash) (int64, error) {
	row, err := s.q.ObjectRead(s.ctx, db.ObjectReadParams{RepoID: s.repoID, Oid: h[:]})
	if err != nil {
		return 0, plumbing.ErrObjectNotFound
	}
	return int64(row.Size), nil
}

func (s *PostgresStorer) AddAlternate(remote string) error { return nil }

// ReferenceStorer
func (s *PostgresStorer) SetReference(ref *plumbing.Reference) error {
	if ref == nil {
		return nil
	}
	if ref.Type() == plumbing.SymbolicReference {
		return s.q.RefSetSymbolic(s.ctx, db.RefSetSymbolicParams{
			RepoID: s.repoID, Name: string(ref.Name()), Target: string(ref.Target()),
		})
	}
	hash := ref.Hash()
	_, err := s.q.RefUpdate(s.ctx, db.RefUpdateParams{
		RepoID: s.repoID, Name: string(ref.Name()), NewOid: hash[:], OldOid: nil, Force: true,
	})
	return err
}

func (s *PostgresStorer) CheckAndSetReference(new, old *plumbing.Reference) error {
	if new == nil {
		return nil
	}
	if new.Type() == plumbing.SymbolicReference {
		return s.q.RefSetSymbolic(s.ctx, db.RefSetSymbolicParams{
			RepoID: s.repoID, Name: string(new.Name()), Target: string(new.Target()),
		})
	}
	var oldOid any
	if old != nil && old.Type() == plumbing.HashReference {
		h := old.Hash()
		oldOid = h[:]
	}
	newHash := new.Hash()
	okRes, err := s.q.RefUpdate(s.ctx, db.RefUpdateParams{
		RepoID: s.repoID,
		Name:   string(new.Name()),
		NewOid: newHash[:],
		OldOid: oldOid,
		Force:  false,
	})
	if err != nil {
		return err
	}
	if ok, _ := okRes.(bool); !ok {
		return storage.ErrReferenceHasChanged
	}
	return nil
}

func (s *PostgresStorer) Reference(n plumbing.ReferenceName) (*plumbing.Reference, error) {
	refs, err := s.q.ListRefsPrealloc(s.ctx, s.repoID)
	if err != nil {
		return nil, err
	}
	for _, r := range refs {
		if r.Name != string(n) {
			continue
		}
		if len(r.Oid) == 20 {
			var h plumbing.Hash
			copy(h[:], r.Oid)
			return plumbing.NewHashReference(n, h), nil
		}
		if r.Symbolic.Valid {
			return plumbing.NewSymbolicReference(n, plumbing.ReferenceName(r.Symbolic.String)), nil
		}
	}
	return nil, plumbing.ErrReferenceNotFound
}

func (s *PostgresStorer) IterReferences() (storer.ReferenceIter, error) {
	refs, err := s.q.ListRefsPrealloc(s.ctx, s.repoID)
	if err != nil {
		return nil, err
	}
	list := make([]*plumbing.Reference, 0, len(refs))
	for _, r := range refs {
		if len(r.Oid) == 20 {
			var h plumbing.Hash
			copy(h[:], r.Oid)
			list = append(list, plumbing.NewHashReference(plumbing.ReferenceName(r.Name), h))
		} else if r.Symbolic.Valid {
			list = append(list, plumbing.NewSymbolicReference(plumbing.ReferenceName(r.Name), plumbing.ReferenceName(r.Symbolic.String)))
		}
	}
	return storer.NewReferenceSliceIter(list), nil
}

var zeroOid20 [20]byte // package-level to avoid per-call allocation in RemoveReference

func (s *PostgresStorer) RemoveReference(n plumbing.ReferenceName) error {
	okRes, err := s.q.RefUpdate(s.ctx, db.RefUpdateParams{
		RepoID: s.repoID, Name: string(n), NewOid: zeroOid20[:], OldOid: nil, Force: true,
	})
	if err != nil {
		return err
	}
	if ok, _ := okRes.(bool); !ok {
		return plumbing.ErrReferenceNotFound
	}
	return nil
}

// RefRow is one ref row from the DB (name, oid, optional symbolic target).
// Used by callers that need a single query and pre-allocated slice.
type RefRow struct {
	Name          string
	Oid           []byte
	Symbolic      string
	SymbolicValid bool
}

// ListRefsRows returns refs from the DB for this repo.
func (s *PostgresStorer) ListRefsRows() ([]RefRow, error) {
	rows, err := s.q.ListRefsPrealloc(s.ctx, s.repoID)
	if err != nil {
		return nil, err
	}
	out := make([]RefRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, RefRow{
			Name:          r.Name,
			Oid:           r.Oid,
			Symbolic:      r.Symbolic.String,
			SymbolicValid: r.Symbolic.Valid,
		})
	}
	return out, nil
}

func (s *PostgresStorer) CountLooseRefs() (int, error) {
	refs, err := s.q.ListRefsPrealloc(s.ctx, s.repoID)
	return len(refs), err
}

func (s *PostgresStorer) PackRefs() error { return nil }

// ShallowStorer
func (s *PostgresStorer) SetShallow(commits []plumbing.Hash) error { return nil }
func (s *PostgresStorer) Shallow() ([]plumbing.Hash, error)        { return nil, nil }

// IndexStorer
func (s *PostgresStorer) SetIndex(idx *index.Index) error { return nil }
func (s *PostgresStorer) Index() (*index.Index, error) {
	return &index.Index{Version: 2}, nil
}

// ConfigStorer
func (s *PostgresStorer) SetConfig(cfg *config.Config) error { return nil }
func (s *PostgresStorer) Config() (*config.Config, error) {
	return config.NewConfig(), nil
}

// ModuleStorer
func (s *PostgresStorer) Module(name string) (storage.Storer, error) {
	return nil, nil
}
