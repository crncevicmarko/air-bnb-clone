package create_accommodation

import "time"

type ReservationByAccommodation1 struct {
	AccoId               string
	HostId               string
	NumberPeople         int
	PriceByPeople        int
	PriceByAccommodation int
	StartDate            time.Time
	EndDate              time.Time
}

type CreateAccommodationCommandType int8

const (
	CreateReservation     CreateAccommodationCommandType = iota // kreiramo rezervaciju
	RollbackReservation                                         // rollbackujemo reservaciju ako nije nesto dobro
	RollbackAccommodation                                       // rollbackujemo akomodaciju ako nije nesto dobro
	ApproveAccommodation
	UnknownCommand
)

type CreateAccommodationCommand struct {
	Accommodation ReservationByAccommodation1
	Type          CreateAccommodationCommandType
}

type CreateAccommodationReplyType int8

const (
	ResevationCreated       CreateAccommodationReplyType = iota // kreirana je rezervacija
	ReservationNotCreated                                       // nije kreirana rezervacija
	ReservationRolledBack                                       // rollbackovana rezervacija
	AccommodationRolledBack                                     // rollbackovana akomodacija
	UnknownReply
)

type CreateAccommodationReply struct {
	Accommodation ReservationByAccommodation1 // saljemo rezervaciju
	Type          CreateAccommodationReplyType
}
