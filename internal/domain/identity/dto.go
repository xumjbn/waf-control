package identity

// dto.go holds the response shapes the React SPA consumes. Keeping these
// separate from the internal Role/User structs lets us evolve the DB
// representation without breaking the frontend contract — and vice versa.
//
// Schema reference: waf-admin/src/mocks/identity.ts (the MSW mock the SPA
// was originally wired against; the live backend response must match it
// field-for-field so usePermission / RolesGrid / UsersList render correctly).

// RoleDTO is the shape returned by /api/v1/identity/roles and embedded
// inside /me and /users responses.
type RoleDTO struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`         // frontend reads canonical key here
	DisplayName string  `json:"display_name"` // 中文显示名
	Description string  `json:"description,omitempty"`
	Modules     Modules `json:"modules"`
	Readonly    bool    `json:"readonly"`
	Color       string  `json:"color,omitempty"`
	UserCount   int     `json:"user_count,omitempty"`
}

// ToDTO converts the internal Role to its wire shape, collapsing the
// "[*]" / "*" wildcard encodings into the Modules union.
func (r Role) ToDTO() RoleDTO {
	mods := Modules{List: r.Permissions}
	if r.IsWildcard() {
		mods = Modules{Wildcard: true}
	}
	key := r.RoleKey
	if key == "" {
		key = r.Name // legacy fallback when 000008 has not been applied
	}
	return RoleDTO{
		ID:          r.ID,
		Name:        key,
		DisplayName: r.Name,
		Description: r.Description,
		Modules:     mods,
		Readonly:    r.Readonly,
		Color:       r.Color,
	}
}

// MeResponse is the GET /me payload — what auth.ts maps into the SPA's
// auth store. The roles array is the load-bearing piece: usePermission
// reads roles[*].name (the key) and roles[*].modules.
type MeResponse struct {
	ID       int64     `json:"id"`
	Username string    `json:"username"`
	RealName string    `json:"real_name,omitempty"`
	Email    string    `json:"email,omitempty"`
	Avatar   string    `json:"avatar,omitempty"`
	Project  string    `json:"project,omitempty"`
	Roles    []RoleDTO `json:"roles"`
}

// UserListItem is one row of GET /api/v1/identity/users — flattened for
// the user list table in NW · 10.
type UserListItem struct {
	ID        int64       `json:"id"`
	Username  string      `json:"username"`
	Email     string      `json:"email,omitempty"`
	RealName  string      `json:"real_name,omitempty"`
	Enabled   bool        `json:"enabled"`
	Avatar    string      `json:"avatar,omitempty"`
	Project   string      `json:"project,omitempty"`
	LastLogin string      `json:"last_login,omitempty"`
	Role      *RoleDigest `json:"role,omitempty"`
}

// RoleDigest is the trimmed role payload embedded in user list rows.
type RoleDigest struct {
	ID    int64  `json:"id"`
	Key   string `json:"key"`   // canonical key
	Name  string `json:"name"`  // 中文显示名
	Color string `json:"color"`
}

// ToUserListItem flattens a User (with its first role) for the NW · 10 list.
func ToUserListItem(u User) UserListItem {
	item := UserListItem{
		ID:       u.ID,
		Username: u.Username,
		Enabled:  u.IsActive,
	}
	if u.Email != nil {
		item.Email = *u.Email
	}
	if u.RealName != nil {
		item.RealName = *u.RealName
	}
	if u.Avatar != nil {
		item.Avatar = *u.Avatar
	} else if len(u.Username) > 0 {
		item.Avatar = string([]rune(u.Username)[0])
	}
	if u.Project != nil {
		item.Project = *u.Project
	}
	if u.LastLogin != nil {
		item.LastLogin = u.LastLogin.Format("2006-01-02 15:04")
	} else {
		item.LastLogin = "—"
	}
	if len(u.Roles) > 0 {
		r := u.Roles[0]
		key := r.RoleKey
		if key == "" {
			key = r.Name
		}
		item.Role = &RoleDigest{ID: r.ID, Key: key, Name: r.Name, Color: r.Color}
	}
	return item
}
