package params

const UserTarifOnline = 1

// Parameters for user business logic
type UserLogicParams struct {
	UserId      uint64 `db:"user_id"`
	TarifId     uint64 `db:"tarif_id"`
	Money       uint64 `db:"money"`
	Vip         bool   `db:"vip"`
	LegalEntity bool   `db:"isLegalEntity"`
	Blocked     bool   `db:"blocked"`
	Push        bool   `db:"push"`
}

func (p *UserLogicParams) CanBeRecorded() bool {
	return (p.TarifId > UserTarifOnline && p.Money > 0) || p.Vip || p.LegalEntity
}
