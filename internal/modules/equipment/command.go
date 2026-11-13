package equipment

import "infradesktop/internal/modules/access"

type Command struct {
	GroupEmpty                                                      bool
	ContactSecretStamp                                              string
	PendingCredential                                               *SealedCredential
	PendingWrapped                                                  []byte
	ContactCredentialRevision                                       uint64
	GroupID, ParentID                                               string
	SaveAnyway                                                      bool
	Contact                                                         *ContactResult
	ContactCode                                                     Code
	Table                                                           string
	TablePreference                                                 *TablePreference
	All                                                             bool
	Import                                                          *ImportInput
	Bulk                                                            *BulkUpdate
	Authority                                                       uint64
	Op                                                              string
	Scope                                                           access.Command
	UID                                                             uint32
	Operation, ID, Name, Address, CredentialID, Alias, User, Secret string
	Transport                                                       Transport
	Role                                                            Role
	Revision                                                        uint64
	Port                                                            int
	Active, Inactive, IncludeInactive, BlockEmpty                   bool
	Cursor, Affected                                                int
	State, Search, Block, Model                                     string
	JobID, Fingerprint                                              string
}
