package store

import "context"

// UserGroupIDs returns the set of group IDs that the given user belongs to.
// Used by permission filtering: a row is readable if user.id ∈ row.read_perms
// OR any groupID(user) ∈ row.read_perms.
func (s *Store) UserGroupIDs(ctx context.Context, userID string) ([]string, error) {
	rows, err := s.Pool.Query(ctx,
		`SELECT group_id FROM users_groups_members WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, nil
}

// UserPrincipals returns userID + all groupIDs the user belongs to. Special
// principals: empty string user → no scoping applied by callers.
func (s *Store) UserPrincipals(ctx context.Context, userID string) ([]string, error) {
	gids, err := s.UserGroupIDs(ctx, userID)
	if err != nil {
		return nil, err
	}
	return append([]string{userID}, gids...), nil
}
