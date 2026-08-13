package character

import "fmt"

func (s *Store) ListPage(page, pageSize int, search string) ([]Character, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 20
	}

	where := "WHERE deleted_at IS NULL"
	args := []interface{}{}
	argIdx := 1

	if search != "" {
		where += fmt.Sprintf(" AND name ILIKE $%d", argIdx)
		args = append(args, "%"+search+"%")
		argIdx++
	}

	var total int
	countQuery := "SELECT COUNT(*) FROM characters " + where
	if err := s.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	query := "SELECT id, name, description, metadata, created_at, updated_at, deleted_at" +
		" FROM characters " + where + " ORDER BY created_at DESC LIMIT $" +
		fmt.Sprintf("%d", argIdx) + " OFFSET $" + fmt.Sprintf("%d", argIdx+1)
	args = append(args, pageSize, offset)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var chars []Character
	for rows.Next() {
		var ch Character
		if err := rows.Scan(&ch.ID, &ch.Name, &ch.Description, &ch.Metadata,
			&ch.CreatedAt, &ch.UpdatedAt, &ch.DeletedAt); err != nil {
			return nil, 0, err
		}
		chars = append(chars, ch)
	}
	if chars == nil {
		chars = []Character{}
	}

	return chars, total, nil
}

