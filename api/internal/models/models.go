// Package models contains the core entity structs used by the store and the
// OData / REST layer. Field naming intentionally mirrors jsreport's entity
// shape (camelCase via JSON tags) so existing jsreport clients can talk to us.
package models

import (
	"encoding/json"
	"time"
)

type User struct {
	ID           string    `json:"_id"`
	Shortid      string    `json:"shortid"`
	Username     string    `json:"username"`
	Email        *string   `json:"email,omitempty"`
	PasswordHash string    `json:"-"`
	IsAdmin      bool      `json:"isAdmin"`
	ReadAll      bool      `json:"readAllPermissions"`
	EditAll      bool      `json:"editAllPermissions"`
	CreatedAt    time.Time `json:"creationDate"`
	UpdatedAt    time.Time `json:"modificationDate"`
}

type APIKey struct {
	ID         string     `json:"_id"`
	UserID     string     `json:"userId"`
	Name       string     `json:"name"`
	KeyPrefix  string     `json:"keyPrefix"`
	Scopes     []string   `json:"scopes"`
	LastUsedAt *time.Time `json:"lastUsedAt,omitempty"`
	ExpiresAt  *time.Time `json:"expiresAt,omitempty"`
	RevokedAt  *time.Time `json:"revokedAt,omitempty"`
	CreatedAt  time.Time  `json:"creationDate"`
	// Raw is only populated on creation, never on read.
	Raw string `json:"key,omitempty"`
}

type Folder struct {
	ID         string    `json:"_id"`
	Shortid    string    `json:"shortid"`
	Name       string    `json:"name"`
	ParentID   *string   `json:"folder,omitempty"`
	OwnerID    *string   `json:"ownerId,omitempty"`
	ReadPerms  []string  `json:"readPermissions"`
	EditPerms  []string  `json:"editPermissions"`
	CreatedAt  time.Time `json:"creationDate"`
	UpdatedAt  time.Time `json:"modificationDate"`
}

type Template struct {
	ID             string          `json:"_id"`
	Shortid        string          `json:"shortid"`
	Name           string          `json:"name"`
	FolderID       *string         `json:"folder,omitempty"`
	Content        string          `json:"content"`
	Engine         string          `json:"engine"`
	Recipe         string          `json:"recipe"`
	Helpers        string          `json:"helpers"`
	CSS            string          `json:"css"`
	PageSize        string         `json:"pageSize"`
	PageOrientation string         `json:"pageOrientation"`
	PageMargin      string         `json:"pageMargin"`
	DataShortid    *string         `json:"dataShortid,omitempty"`
	Scripts        json.RawMessage `json:"scripts"`
	Chrome         json.RawMessage `json:"chrome,omitempty"`
	WeasyPrint     json.RawMessage `json:"weasyprint,omitempty"`
	Docx           json.RawMessage `json:"docx,omitempty"`
	Xlsx           json.RawMessage `json:"xlsx,omitempty"`
	Pptx           json.RawMessage `json:"pptx,omitempty"`
	PdfOperations  json.RawMessage `json:"pdfOperations,omitempty"`
	// ReportRetentionDays controls how long generated reports from this
	// template are kept before the background sweeper deletes them. 0 means
	// "keep forever". Default 30.
	ReportRetentionDays int        `json:"reportRetentionDays"`
	IsPublic       bool            `json:"isPublic"`
	OwnerID        *string         `json:"ownerId,omitempty"`
	ReadPerms      []string        `json:"readPermissions"`
	EditPerms      []string        `json:"editPermissions"`
	CreatedAt      time.Time       `json:"creationDate"`
	UpdatedAt      time.Time       `json:"modificationDate"`
}

type Asset struct {
	ID                   string    `json:"_id"`
	Shortid              string    `json:"shortid"`
	Name                 string    `json:"name"`
	FolderID             *string   `json:"folder,omitempty"`
	BlobKey              *string   `json:"-"`
	InlineContent        *string   `json:"content,omitempty"`
	MimeType             *string   `json:"mimeType,omitempty"`
	SizeBytes            int64     `json:"size"`
	IsSharedHelper       bool      `json:"isSharedHelper"`
	SharedHelpersScope   *string   `json:"sharedHelpersScope,omitempty"`
	Link                 *string   `json:"link,omitempty"`
	OwnerID              *string   `json:"ownerId,omitempty"`
	CreatedAt            time.Time `json:"creationDate"`
	UpdatedAt            time.Time `json:"modificationDate"`
}

type Script struct {
	ID        string    `json:"_id"`
	Shortid   string    `json:"shortid"`
	Name      string    `json:"name"`
	FolderID  *string   `json:"folder,omitempty"`
	Content   string    `json:"content"`
	Scope     string    `json:"scope"`
	IsGlobal  bool      `json:"isGlobal"`
	OwnerID   *string   `json:"ownerId,omitempty"`
	CreatedAt time.Time `json:"creationDate"`
	UpdatedAt time.Time `json:"modificationDate"`
}

type DataItem struct {
	ID        string          `json:"_id"`
	Shortid   string          `json:"shortid"`
	Name      string          `json:"name"`
	FolderID  *string         `json:"folder,omitempty"`
	DataJSON  json.RawMessage `json:"dataJson"`
	OwnerID   *string         `json:"ownerId,omitempty"`
	CreatedAt time.Time       `json:"creationDate"`
	UpdatedAt time.Time       `json:"modificationDate"`
}

type Component struct {
	ID        string    `json:"_id"`
	Shortid   string    `json:"shortid"`
	Name      string    `json:"name"`
	FolderID  *string   `json:"folder,omitempty"`
	Content   string    `json:"content"`
	Engine    string    `json:"engine"`
	Helpers   string    `json:"helpers"`
	OwnerID   *string   `json:"ownerId,omitempty"`
	CreatedAt time.Time `json:"creationDate"`
	UpdatedAt time.Time `json:"modificationDate"`
}

type Report struct {
	ID              string    `json:"_id"`
	Shortid         string    `json:"shortid"`
	Name            *string   `json:"name,omitempty"`
	TemplateShortid *string   `json:"templateShortid,omitempty"`
	State           string    `json:"state"`
	Error           *string   `json:"error,omitempty"`
	MimeType        *string   `json:"contentType,omitempty"`
	BlobKey         *string   `json:"-"`
	SizeBytes       int64     `json:"size"`
	IsPublic        bool      `json:"public"`
	TaskID          *string   `json:"taskId,omitempty"`
	OwnerID         *string   `json:"ownerId,omitempty"`
	CreatedAt       time.Time `json:"creationDate"`
}

type Schedule struct {
	ID              string     `json:"_id"`
	Shortid         string     `json:"shortid"`
	Name            string     `json:"name"`
	Cron            string     `json:"cron"`
	TemplateShortid string     `json:"templateShortid"`
	Enabled         bool       `json:"enabled"`
	State           string     `json:"state"`
	NextRun         *time.Time `json:"nextRun,omitempty"`
	OwnerID         *string    `json:"ownerId,omitempty"`
	CreatedAt       time.Time  `json:"creationDate"`
	UpdatedAt       time.Time  `json:"modificationDate"`
}

type Profile struct {
	ID              string     `json:"_id"`
	Shortid         string     `json:"shortid"`
	TemplateShortid *string    `json:"templateShortid,omitempty"`
	State           string     `json:"state"`
	Mode            string     `json:"mode"`
	Error           *string    `json:"error,omitempty"`
	BlobKey         *string    `json:"-"`
	TimeoutMs       int        `json:"timeout"`
	StartedAt       time.Time  `json:"timestamp"`
	FinishedAt      *time.Time `json:"finishedOn,omitempty"`
	OwnerID         *string    `json:"ownerId,omitempty"`
}

type Setting struct {
	ID        string          `json:"_id"`
	Key       string          `json:"key"`
	Value     json.RawMessage `json:"value"`
	UpdatedAt time.Time       `json:"modificationDate"`
}

type Version struct {
	ID        string    `json:"_id"`
	Shortid   string    `json:"shortid"`
	Message   string    `json:"message"`
	BlobKey   string    `json:"-"`
	AuthorID  *string   `json:"authorId,omitempty"`
	CreatedAt time.Time `json:"creationDate"`
}

type Tag struct {
	ID        string    `json:"_id"`
	Shortid   string    `json:"shortid"`
	Name      string    `json:"name"`
	Color     string    `json:"color"`
	CreatedAt time.Time `json:"creationDate"`
}

type UsersGroup struct {
	ID        string    `json:"_id"`
	Shortid   string    `json:"shortid"`
	Name      string    `json:"name"`
	IsAdmin   bool      `json:"isAdmin"`
	UserIDs   []string  `json:"users"`
	CreatedAt time.Time `json:"creationDate"`
	UpdatedAt time.Time `json:"modificationDate"`
}
