package main

import (
	"log"

	events "github.com/vukasinc25/fst-airbnb/utility/saga/create_accommodation"
	saga "github.com/vukasinc25/fst-airbnb/utility/saga/messaging"
)

type CreateResrvationCommandHandler struct {
	repo              *ReservationRepo // videcemo da li ce da se resi ako postavimo sve lepo u main.go #0 13.87 handlers/accommodationHandle.go:9:21: undefined: AccoRepo #0 13.87 handlers/accommodationHandle.go:14:39: undefined: AccoRep
	replyPublisher    saga.Publisher
	commandSubscriber saga.Subscriber
}

func NewCreateReservationCommandHandler(r *ReservationRepo, publisher saga.Publisher, subscriber saga.Subscriber) (*CreateResrvationCommandHandler, error) {
	o := &CreateResrvationCommandHandler{
		repo:              r,
		replyPublisher:    publisher,  // order.create.reply
		commandSubscriber: subscriber, // order.create.command
	}
	err := o.commandSubscriber.Subscribe(o.handle)
	if err != nil {
		return nil, err
	}
	return o, nil
}

func (handler *CreateResrvationCommandHandler) handle(command *events.CreateAccommodationCommand) {
	log.Println("Udjosmo")
	log.Println("Type: ", command.Type)
	reservation := ReservationByAccommodation{ // ovde
		AccoId:               command.Accommodation.AccoId,
		HostId:               command.Accommodation.HostId,
		NumberPeople:         command.Accommodation.NumberPeople,
		PriceByPeople:        command.Accommodation.PriceByPeople,
		PriceByAccommodation: command.Accommodation.PriceByAccommodation,
		StartDate:            command.Accommodation.StartDate, // vidi da li prosledjuje dobar datum
		EndDate:              command.Accommodation.EndDate,   // vidi da li prosledjuje dobar datum
	}

	reply := events.CreateAccommodationReply{Accommodation: command.Accommodation} // valjda je dobro

	switch command.Type {
	case events.CreateReservation:
		log.Println("Od koga u reservaciji: ", "CreateReservation")
		log.Println("Reservation: ", reservation)
		err := handler.repo.InsertReservationByAcco(&reservation)
		if err != nil {
			reply.Type = events.ReservationNotCreated
			break
		}

		reply.Type = events.ResevationCreated
	case events.RollbackReservation:
		log.Println("Od koga u reservaciji: ", "RollbackReservation")
		err := handler.repo.UpdateReservationByAcco("", "", "")
		if err != nil {
			return
		}
		reply.Type = events.ReservationRolledBack
	default:
		reply.Type = events.UnknownReply
	}

	if reply.Type != events.UnknownReply {
		_ = handler.replyPublisher.Publish(reply)
	}
}
