package main

import (
	"log"

	events "github.com/vukasinc25/fst-airbnb/utility/saga/create_accommodation"
	saga "github.com/vukasinc25/fst-airbnb/utility/saga/messaging"
)

type CreateAccomodationCommandHandler struct {
	db                *AccoRepo // videcemo da li ce da se resi ako postavimo sve lepo u main.go #0 13.87 handlers/accommodationHandle.go:9:21: undefined: AccoRepo #0 13.87 handlers/accommodationHandle.go:14:39: undefined: AccoRep
	replyPublisher    saga.Publisher
	commandSubscriber saga.Subscriber
}

func NewCreateAccommodationCommandHandler(db *AccoRepo, publisher saga.Publisher, subscriber saga.Subscriber) (*CreateAccomodationCommandHandler, error) {
	o := &CreateAccomodationCommandHandler{
		db:                db,
		replyPublisher:    publisher,  // order.create.reply
		commandSubscriber: subscriber, // order.create.command
	}
	err := o.commandSubscriber.Subscribe(o.handle)
	if err != nil {
		return nil, err
	}
	return o, nil
}

func (handler *CreateAccomodationCommandHandler) handle(command *events.CreateAccommodationCommand) {
	var id = command.Accommodation.AccoId

	// mora da se podesi na akomodaciju ili samo treba da se prosledi id

	reply := events.CreateAccommodationReply{Accommodation: command.Accommodation} // valjda je dobro

	switch command.Type {
	case events.ApproveAccommodation:
		log.Println("Od koga u akomodacije: ", "ReservationCreated")
		err := handler.db.UpdateAccommodation(id) // vidi kako se zove metoda
		if err != nil {
			return
		}
		reply.Type = events.UnknownReply
	case events.RollbackAccommodation:
		log.Println("Od koga u akomodacije: ", "RollbackAccommodation")
		err := handler.db.DeleteById(id) // vidi kako se zove metoda
		if err != nil {
			return
		}
		reply.Type = events.AccommodationRolledBack
	default:
		log.Println("Od koga u akomodacije: ", "UnknownReply")
		reply.Type = events.UnknownReply
	}

	log.Println("Handling command: %v", command)
	log.Println("Publishing reply: %v", reply)

	if reply.Type != events.UnknownReply {
		_ = handler.replyPublisher.Publish(reply)
	}
}
