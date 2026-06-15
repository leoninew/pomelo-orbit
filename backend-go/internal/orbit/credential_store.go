package orbit

import (
	"context"
	"fmt"
	"strings"

	"backend/internal/db"
)

func (s Store) ListCredentials(ctx context.Context, projectId string, page int, perPage int, search string) (Page[Credential], error) {
	page, perPage = NormalizePage(page, perPage)
	where, args := credentialWhere(projectId, search)
	var total int
	if err := s.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM credential`+where, args...); err != nil {
		return Page[Credential]{}, fmt.Errorf("count credentials: %w", err)
	}
	args = append(args, perPage, (page-1)*perPage)
	var items []Credential
	err := s.db.SelectContext(ctx, &items, `SELECT id, project_id, name, type, encrypted_data, created_at FROM credential`+where+` ORDER BY id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return Page[Credential]{}, fmt.Errorf("list credentials: %w", err)
	}
	return Page[Credential]{Items: items, Total: total, Page: page, PerPage: perPage}, nil
}

func (s Store) CredentialByName(ctx context.Context, projectId string, name string) (Credential, error) {
	var credential Credential
	err := s.db.GetContext(ctx, &credential, `SELECT id, project_id, name, type, encrypted_data, created_at FROM credential WHERE project_id = ? AND name = ?`, strings.TrimSpace(projectId), strings.TrimSpace(name))
	if err != nil {
		return Credential{}, fmt.Errorf("load credential by name %s: %w", name, err)
	}
	return credential, nil
}

func (s Store) CreateCredential(ctx context.Context, credential Credential) error {
	_, err := s.db.ExecContext(ctx, fmt.Sprintf(`INSERT INTO credential (id, project_id, name, type, encrypted_data, created_at) VALUES (?, ?, ?, ?, ?, %s)`, db.NowExpr(s.driver)), credential.Id, credential.ProjectId, credential.Name, credential.Type, credential.EncryptedData)
	if err != nil {
		return fmt.Errorf("create credential %s: %w", credential.Name, err)
	}
	return nil
}

func (s Store) UpdateCredential(ctx context.Context, credential Credential) error {
	_, err := s.db.ExecContext(ctx, `UPDATE credential SET name = ?, encrypted_data = ? WHERE id = ?`, credential.Name, credential.EncryptedData, credential.Id)
	if err != nil {
		return fmt.Errorf("update credential %s: %w", credential.Id, err)
	}
	return nil
}

func (s Store) DeleteCredential(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM credential WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete credential %s: %w", id, err)
	}
	return nil
}

func (s Store) CredentialReferencedByRepositories(ctx context.Context, projectId string, credentialId string) (bool, error) {
	var count int
	if err := s.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM repository WHERE project_id = ? AND git_credential_id = ?`, strings.TrimSpace(projectId), credentialId); err != nil {
		return false, fmt.Errorf("check credential references %s: %w", credentialId, err)
	}
	return count > 0, nil
}

func credentialWhere(projectId string, search string) (string, []any) {
	clauses := []string{"project_id = ?"}
	args := []any{strings.TrimSpace(projectId)}
	search = strings.TrimSpace(search)
	if search != "" {
		like := "%" + search + "%"
		clauses = append(clauses, "(name LIKE ? OR type LIKE ?)")
		args = append(args, like, like)
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}
