package main

import (
	"log"

	events "github.com/vukasinc25/fst-airbnb/utility/saga/create_accommodation" // ovde treba da se promeni za svoju potrebu
	saga "github.com/vukasinc25/fst-airbnb/utility/saga/messaging"
)

type CreateAccommodationOrchestrator struct {
	commandPublisher saga.Publisher
	replySubscriber  saga.Subscriber
}

func NewCreateAccommodationOrchestrator(publisher saga.Publisher, subscriber saga.Subscriber) (*CreateAccommodationOrchestrator, error) {
	log.Println("OrchestratorPublisher: ", publisher)
	log.Println("OrchestratorSubscriber: ", subscriber)
	o := &CreateAccommodationOrchestrator{
		commandPublisher: publisher,  // order.create.command
		replySubscriber:  subscriber, // order.create.reply
	}
	log.Println("CreateAccommodationOrchestrator: ", o)
	err := o.replySubscriber.Subscribe(o.handle)
	if err != nil {
		log.Println("Error trying Subscribe: ", err)
		return nil, err
	}
	return o, nil
}

func (o *CreateAccommodationOrchestrator) Start(reservation *ReservationByAccommodation1) error {
	// kreiramo dogadjaj za kreiranje proizvoda koji ce mo da prosledimo ostalim servisima tipa je updateInventori zato sto zelimo da prosirimo skladiste
	event := &events.CreateAccommodationCommand{
		Type: events.CreateReservation,
		Accommodation: events.ReservationByAccommodation1{
			AccoId:               reservation.AccoId,
			HostId:               reservation.HostId,
			NumberPeople:         reservation.NumberPeople,
			PriceByPeople:        reservation.PriceByPeople,
			PriceByAccommodation: reservation.PriceByAccommodation,
			StartDate:            reservation.StartDate,
			EndDate:              reservation.EndDate,
		},
	}
	return o.commandPublisher.Publish(event) // kada iteriramo kroz sv proizvode iz narudzbine koju smo hteli da kreiramo publisujemo taj dogadjaj koji sadrzi tip komande i samu narudzbinu
}

func (o *CreateAccommodationOrchestrator) handle(reply *events.CreateAccommodationReply) {
	log.Println("Orkestrator")
	command := events.CreateAccommodationCommand{Accommodation: reply.Accommodation} // u promenjivu command prosledimo komandu za kreiranje narudzbine i definisemo samo narudzbinu ne i tip u ovom slucaju odgovor koji dobijemo kao primajuci parametar
	command.Type = o.nextCommandType(reply.Type)                                     // definisemo i tip komande koji je tip odgovora koji dobijemo kao primajuci parametar
	if command.Type != events.UnknownCommand {                                       // proveravamo ako tip odgovora nije nepoznat publisujemo komandu koju smo kreirali na osnovu odgovora koji smo dobili
		_ = o.commandPublisher.Publish(command)
	}
}

func (o *CreateAccommodationOrchestrator) nextCommandType(reply events.CreateAccommodationReplyType) events.CreateAccommodationCommandType {
	switch reply {
	case events.ResevationCreated:
		log.Println("Od koga: ", "ResevationCreated")
		return events.ApproveAccommodation
	case events.ReservationNotCreated:
		log.Println("Od koga: ", "ReservationNotCreated")
		return events.RollbackAccommodation
	case events.ReservationRolledBack:
		log.Println("Od koga: ", "ReservationRolledBack")
		return events.RollbackAccommodation
	default:
		log.Println("Od koga: ", "UnknownCommand")
		return events.UnknownCommand
	}
}
