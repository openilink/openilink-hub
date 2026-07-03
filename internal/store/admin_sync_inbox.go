package store

type AdminSyncInboxStore interface {
	CreateAdminSyncInboxEvent(eventID string) (bool, error)
	HasAdminSyncInboxEvent(eventID string) (bool, error)
}
