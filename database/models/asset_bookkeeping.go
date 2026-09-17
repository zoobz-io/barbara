package models

import "time"

// AssetFolderStat is the bookkeeping row for one folder path of an app's asset
// tree: the rollup of every asset beneath it. Path is "" for the root, else
// the folder without a trailing slash. Explicit marks a folder created on
// purpose, which survives at zero count; every other row exists only while a
// key sits under it. Never authoritative — the bucket is.
type AssetFolderStat struct {
	CreatedAt     time.Time  `json:"created_at" db:"created_at" default:"now()"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at" default:"now()"`
	LastWrittenAt *time.Time `json:"last_written_at,omitempty" db:"last_written_at"`
	ID            string     `json:"id" db:"id" constraints:"primarykey"`
	TenantID      string     `json:"tenant_id" db:"tenant_id" constraints:"notnull"`
	AppID         string     `json:"app_id" db:"app_id" constraints:"notnull"`
	Path          string     `json:"path" db:"path" constraints:"notnull"`
	Count         int64      `json:"count" db:"count" constraints:"notnull"`
	Bytes         int64      `json:"bytes" db:"bytes" constraints:"notnull"`
	Explicit      bool       `json:"explicit" db:"explicit" constraints:"notnull"`
}

// GetID returns the row's primary key.
func (f AssetFolderStat) GetID() string { return f.ID }

// Clone returns a deep copy.
func (f AssetFolderStat) Clone() AssetFolderStat {
	c := f
	if f.LastWrittenAt != nil {
		t := *f.LastWrittenAt
		c.LastWrittenAt = &t
	}
	return c
}

// AssetKindStat is the app-level rollup for one media family. Kind "" is the
// all-kinds total. Root only: folders never carry a per-kind split.
type AssetKindStat struct {
	LastWrittenAt *time.Time `json:"last_written_at,omitempty" db:"last_written_at"`
	ID            string     `json:"id" db:"id" constraints:"primarykey"`
	TenantID      string     `json:"tenant_id" db:"tenant_id" constraints:"notnull"`
	AppID         string     `json:"app_id" db:"app_id" constraints:"notnull"`
	Kind          Kind       `json:"kind" db:"kind" constraints:"notnull"`
	Count         int64      `json:"count" db:"count" constraints:"notnull"`
	Bytes         int64      `json:"bytes" db:"bytes" constraints:"notnull"`
}

// GetID returns the row's primary key.
func (k AssetKindStat) GetID() string { return k.ID }

// Clone returns a deep copy.
func (k AssetKindStat) Clone() AssetKindStat {
	c := k
	if k.LastWrittenAt != nil {
		t := *k.LastWrittenAt
		c.LastWrittenAt = &t
	}
	return c
}

// AssetDayStat is one UTC day of writes for one kind ("" = all kinds), keyed
// on the objects' last-modified time. Windows are sums over these rows.
type AssetDayStat struct {
	Day      time.Time `json:"day" db:"day" constraints:"notnull"`
	ID       string    `json:"id" db:"id" constraints:"primarykey"`
	TenantID string    `json:"tenant_id" db:"tenant_id" constraints:"notnull"`
	AppID    string    `json:"app_id" db:"app_id" constraints:"notnull"`
	Kind     Kind      `json:"kind" db:"kind" constraints:"notnull"`
	Count    int64     `json:"count" db:"count" constraints:"notnull"`
	Bytes    int64     `json:"bytes" db:"bytes" constraints:"notnull"`
}

// GetID returns the row's primary key.
func (d AssetDayStat) GetID() string { return d.ID }

// Clone returns a copy.
func (d AssetDayStat) Clone() AssetDayStat { return d }

// AssetBookkeeping records when an app's bookkeeping was last rebuilt from the
// bucket — the "as of" shown beside the numbers.
type AssetBookkeeping struct {
	ComputedAt time.Time `json:"computed_at" db:"computed_at" constraints:"notnull"`
	ID         string    `json:"id" db:"id" constraints:"primarykey"`
	TenantID   string    `json:"tenant_id" db:"tenant_id" constraints:"notnull"`
	AppID      string    `json:"app_id" db:"app_id" constraints:"notnull"`
}

// GetID returns the row's primary key.
func (b AssetBookkeeping) GetID() string { return b.ID }

// Clone returns a copy.
func (b AssetBookkeeping) Clone() AssetBookkeeping { return b }

// AssetStats is the app-level read: the root folder's rollup, the per-kind
// breakdown (all-kinds row excluded), the daily series oldest first, and when
// the bookkeeping was last rebuilt (nil until the first rebuild). Root is
// never nil; an app with no assets gets a zero row.
type AssetStats struct {
	ComputedAt *time.Time
	Root       *AssetFolderStat
	Kinds      []*AssetKindStat
	Days       []*AssetDayStat
}
