package models

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Create necessary tables
	createTablesSQL := `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		username TEXT UNIQUE NOT NULL,
		email TEXT UNIQUE NOT NULL,
		password TEXT NOT NULL,
		full_name TEXT,
		first_name TEXT,
		last_name TEXT,
		date_of_birth TEXT,
		age INTEGER,
		gender TEXT,
		bio TEXT,
		profile_picture TEXT,
		cover_photo TEXT,
		is_private BOOLEAN DEFAULT FALSE,
		role TEXT DEFAULT 'member',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS groups (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT,
		creator_id TEXT NOT NULL,
		cover_photo TEXT,
		privacy TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (creator_id) REFERENCES users(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS group_members (
		id TEXT PRIMARY KEY,
		group_id TEXT NOT NULL,
		user_id TEXT NOT NULL,
		role TEXT NOT NULL,
		status TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(group_id, user_id),
		FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS group_posts (
		id TEXT PRIMARY KEY,
		group_id TEXT NOT NULL,
		user_id TEXT NOT NULL,
		content TEXT NOT NULL,
		image TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);
	`

	_, err = db.Exec(createTablesSQL)
	if err != nil {
		t.Fatalf("Failed to create tables: %v", err)
	}

	return db
}

func TestGroupPostDeletionPermissions(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userService := NewUserService(db)
	groupService := NewGroupService(db)
	groupMemberService := NewGroupMemberService(db)
	groupPostService := NewGroupPostService(db)

	// Create users
	creator := &User{Username: "creator", Email: "creator@example.com", Password: "password"}
	admin := &User{Username: "admin", Email: "admin@example.com", Password: "password"}
	member := &User{Username: "member", Email: "member@example.com", Password: "password"}
	otherUser := &User{Username: "other", Email: "other@example.com", Password: "password"}

	err := userService.Create(creator)
	if err != nil {
		t.Fatalf("Failed to create creator user: %v", err)
	}
	err = userService.Create(admin)
	if err != nil {
		t.Fatalf("Failed to create admin user: %v", err)
	}
	err = userService.Create(member)
	if err != nil {
		t.Fatalf("Failed to create member user: %v", err)
	}
	err = userService.Create(otherUser)
	if err != nil {
		t.Fatalf("Failed to create other user: %v", err)
	}

	// Create a group with creator as its creator
	group := &Group{Name: "Test Group", CreatorID: creator.ID, Privacy: GroupPrivacyPublic}
	err = groupService.Create(group)
	if err != nil {
		t.Fatalf("Failed to create group: %v", err)
	}

	// Add admin and member to the group
	adminMember := &GroupMember{GroupID: group.ID, UserID: admin.ID, Role: GroupMemberRoleAdmin, Status: GroupMemberStatusAccepted}
	err = groupMemberService.Create(adminMember)
	if err != nil {
		t.Fatalf("Failed to add admin member: %v", err)
	}

	regularMember := &GroupMember{GroupID: group.ID, UserID: member.ID, Role: GroupMemberRoleMember, Status: GroupMemberStatusAccepted}
	err = groupMemberService.Create(regularMember)
	if err != nil {
		t.Fatalf("Failed to add regular member: %v", err)
	}

	// Test Cases
	tests := []struct {
		name           string
		postAuthor     *User
		callerID       string
		expectedError  bool
		expectedErrMsg string
	}{
		// Creator permissions
		{"Creator deletes own post", creator, creator.ID, false, ""},
		{"Creator deletes admin's post", admin, creator.ID, false, ""},
		{"Creator deletes member's post", member, creator.ID, false, ""},

		// Admin permissions
		{"Admin deletes own post", admin, admin.ID, false, ""},
		{"Admin deletes member's post", member, admin.ID, false, ""},
		{"Admin tries to delete creator's post", creator, admin.ID, true, "admin cannot delete the group creator's post"},

		// Member permissions
		{"Member deletes own post", member, member.ID, false, ""},
		{"Member tries to delete creator's post", creator, member.ID, true, "not authorized to delete this post"},
		{"Member tries to delete admin's post", admin, member.ID, true, "not authorized to delete this post"},

		// Other user permissions
		{"Other user tries to delete creator's post", creator, otherUser.ID, true, "user is not an active member of this group"},
		{"Other user tries to delete admin's post", admin, otherUser.ID, true, "user is not an active member of this group"},
		{"Other user tries to delete member's post", member, otherUser.ID, true, "user is not an active member of this group"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new post for each test case to ensure isolation
			postToDelete := &GroupPost{GroupID: group.ID, UserID: tt.postAuthor.ID, Content: tt.postAuthor.Username + "'s post"}
			err := groupPostService.Create(postToDelete)
			if err != nil {
				t.Fatalf("Failed to create post for test case %s: %v", tt.name, err)
			}

			err = groupPostService.Delete(postToDelete.ID, tt.callerID)

			if tt.expectedError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if err.Error() != tt.expectedErrMsg {
					t.Errorf("Expected error message '%s' but got '%s'", tt.expectedErrMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}
