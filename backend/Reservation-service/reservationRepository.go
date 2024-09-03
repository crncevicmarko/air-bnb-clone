package main

import (
	"context"
	"errors"
	"fmt"
	"go.opentelemetry.io/otel/trace"

	// "log"
	"os"
	"time"

	"github.com/gocql/gocql"
	log "github.com/sirupsen/logrus"
)

type ReservationRepo struct {
	session *gocql.Session
	logger  *log.Logger
	tracer  trace.Tracer
}

func New(logger *log.Logger, tracer trace.Tracer) (*ReservationRepo, error) {
	db := os.Getenv("CASS_DB")

	cluster := gocql.NewCluster(db)
	cluster.Keyspace = "system"
	cluster.Timeout = time.Second * 55
	session, err := cluster.CreateSession()
	if err != nil {
		logger.Println(err)
		return nil, err
	}

	err = session.Query(
		fmt.Sprintf(`CREATE KEYSPACE IF NOT EXISTS %s
					WITH replication = {
						'class' : 'SimpleStrategy',
						'replication_factor' : %d
					}`, "reservation", 1)).Exec()
	if err != nil {
		logger.Println(err)
	}
	session.Close()

	cluster.Keyspace = "reservation"
	cluster.Consistency = gocql.One
	session, err = cluster.CreateSession()
	if err != nil {
		logger.Println(err)
		return nil, err
	}

	return &ReservationRepo{
		session: session,
		logger:  logger,
		tracer:  tracer,
	}, nil
}

func (rs *ReservationRepo) CloseSession() {
	rs.session.Close()
}

// func (pr *ReservationRepo) Ping() {
// 	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
// 	defer cancel()

// 	// Check connection -> if no error, connection is established
// 	err := pr.cli.Ping(ctx, readpref.Primary())
// 	if err != nil {
// 		pr.logger.Println(err)
// 	}

// 	// Print available databases
// 	databases, err := pr.cli.ListDatabaseNames(ctx, bson.M{})
// 	if err != nil {
// 		pr.logger.Println(err)
// 	}
// 	fmt.Println(databases)
// }

func (rs *ReservationRepo) CreateTables() {
	//RESERVATION BY ACCO
	err := rs.session.Query(
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s
						(reservation_id UUID, acco_id text, host_id text, numberPeople int, priceByPeople int, priceByAcoommodation int,
						startDate date, endDate date, isDeleted boolean,
						PRIMARY KEY ((reservation_id, acco_id), startDate))
						WITH CLUSTERING ORDER BY (startDate DESC)`,
			"reservations_by_acco")).Exec()
	if err != nil {
		rs.logger.Println(err)
	}
	err = rs.session.Query(
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s 
						(reservation_id UUID, acco_id text, host_id text, numberPeople int, priceByPeople int, priceByAcoommodation int,
						startDate date, endDate date,
						PRIMARY KEY ((acco_id), reservation_id, startDate))
						WITH CLUSTERING ORDER BY (reservation_id ASC, startDate DESC)`,
			"reservations_by_acco1")).Exec()
	if err != nil {
		rs.logger.Println(err)
	}

	err = rs.session.Query(
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s 
						(reservation_id UUID, acco_id text, host_id text, numberPeople int, priceByPeople int, priceByAcoommodation int,
						startDate date, endDate date,
						PRIMARY KEY ((host_id), reservation_id, startDate))
						WITH CLUSTERING ORDER BY (reservation_id ASC, startDate DESC)`,
			"reservations_by_acco2")).Exec()
	if err != nil {
		rs.logger.Println(err)
	}

	//RESERVATION BY GUEST
	err = rs.session.Query(
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s 
					(user_id text, reservation_id text, acco_id text, price int, 
						begin_reservation_date date, numberOfPeople int, end_reservation_date date,
					PRIMARY KEY ((user_id), reservation_id, acco_id, begin_reservation_date, end_reservation_date, price))
					WITH CLUSTERING ORDER BY (reservation_id ASC, acco_id ASC, begin_reservation_date ASC, end_reservation_date ASC, price ASC)`,
			"reservations_by_user")).Exec()
	if err != nil {
		rs.logger.Println(err)
	}
	// err = rs.session.Query(
	// 	fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s
	// 				(user_id text, reservation_id text, acco_id text, price int,
	// 					begin_reservation_date date, numberOfPeople int, end_reservation_date date, isDeleted boolean,
	// 				PRIMARY KEY ((user_id, reservation_id, acco_id, begin_reservation_date, end_reservation_date), price))
	// 				WITH CLUSTERING ORDER BY (price ASC)`,
	// 		"reservations_by_user")).Exec()
	// if err != nil {
	// 	rs.logger.Println(err)
	// }

	// err = rs.session.Query(
	// 	fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s
	// 				(user_id text, reservation_id text, acco_id text, price int,
	// 					begin_reservation_date date, numberOfPeople int, end_reservation_date date,
	// 				PRIMARY KEY ((user_id), reservation_id, acco_id, begin_reservation_date, end_reservation_date, price))
	// 				WITH CLUSTERING ORDER BY (reservation_id ASC, acco_id ASC, begin_reservation_date ASC, end_reservation_date ASC, price ASC)`,
	// 		"reservations_test1")).Exec()
	// if err != nil {
	// 	rs.logger.Println(err)
	// }

	//FIND RESERVATION DATES FOR ACCO
	err = rs.session.Query(
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s 
					(accommodation_id text, begin_reservation_date date, end_reservation_date date,
					PRIMARY KEY (accommodation_id, begin_reservation_date, end_reservation_date))`,
			"reservations_dates_by_acco_id")).Exec()
	if err != nil {
		rs.logger.Println(err)
	}

	//SEARCH - START AND END DATE
	err = rs.session.Query(
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s 
					(id UUID, accommodation_id text, begin_reservation_date date, end_reservation_date date, 
					PRIMARY KEY ((begin_reservation_date, end_reservation_date), id))`,
			"reservations_dates_by_date")).Exec()
	if err != nil {
		rs.logger.Println(err)
	}
}

// -------Reservation By Accommodation-------//
func (rs *ReservationRepo) GetReservationsByAcco(acco_id string, ctx context.Context) (ReservationsByAccommodation, error) {
	ctx, span := rs.tracer.Start(ctx, "NotificationRepo.GetReservationsByAcco")
	defer span.End()

	scanner := rs.session.Query(`SELECT reservation_id, acco_id, startDate, endDate, host_id, numberPeople, priceByAcoommodation, priceByPeople FROM reservations_by_acco1 WHERE acco_id = ?;`,
		acco_id).Iter().Scanner() // lista
	var reservations ReservationsByAccommodation
	for scanner.Next() {
		var res ReservationByAccommodation
		err := scanner.Scan(&res.ReservationId, &res.AccoId, &res.StartDate, &res.EndDate, &res.HostId, &res.NumberPeople, &res.PriceByAccommodation, &res.PriceByPeople)
		if err != nil {
			rs.logger.Println("Cant 1", err)
			return nil, err
		}
		reservations = append(reservations, &res)
	}
	if err := scanner.Err(); err != nil {
		rs.logger.Println("Cant 2", err)
		return nil, err
	}
	log.Println(reservations)
	return reservations, nil
}

func (rs *ReservationRepo) InsertReservationByAcco(resAcco *ReservationByAccommodation) error {
	//ctx, span := rs.tracer.Start(ctx, "NotificationRepo.InsertReservationByAcco")
	//defer span.End()

	overlap, err := rs.CheckOverlap1(resAcco.AccoId, resAcco.StartDate, resAcco.EndDate)
	if err != nil {
		return err
	}

	if overlap {
		return errors.New("overlap detected: Cannot insert overlapping date range")
	}

	reservationId, _ := gocql.RandomUUID()
	err = rs.session.Query(
		`INSERT INTO reservations_by_acco1 (reservation_id, acco_id, host_id, numberPeople, priceByPeople, priceByAcoommodation,
			startDate, endDate) VALUES 
		(?, ?, ?, ?, ?, ?, ?, ?);`,
		reservationId, resAcco.AccoId, resAcco.HostId, resAcco.NumberPeople, resAcco.PriceByPeople, resAcco.PriceByAccommodation,
		resAcco.StartDate, resAcco.EndDate).Exec()
	if err != nil {
		rs.logger.Println(err)
		return err
	}
	err = rs.session.Query(
		`INSERT INTO reservations_by_acco2 (reservation_id, acco_id, host_id, numberPeople, priceByPeople, priceByAcoommodation,
			startDate, endDate) VALUES 
		(?, ?, ?, ?, ?, ?, ?, ?);`,
		reservationId, resAcco.AccoId, resAcco.HostId, resAcco.NumberPeople, resAcco.PriceByPeople, resAcco.PriceByAccommodation,
		resAcco.StartDate, resAcco.EndDate).Exec()
	if err != nil {
		rs.logger.Println(err)
		return err
	}

	//Insert into second table
	err = rs.session.Query(
		`INSERT INTO reservations_dates_by_acco_id (accommodation_id, begin_reservation_date, end_reservation_date)
		VALUES (?, ?, ?);`,
		resAcco.AccoId, resAcco.StartDate, resAcco.EndDate).Exec()
	if err != nil {
		rs.logger.Println(err)
		return err
	}
	log.Println("Insert prosao")

	//Insert into third table
	reservationId, _ = gocql.RandomUUID()
	err = rs.session.Query(
		`INSERT INTO reservations_dates_by_date (id, accommodation_id, begin_reservation_date, end_reservation_date) 
		VALUES (?, ?, ?, ?);`,
		reservationId, resAcco.AccoId, resAcco.StartDate, resAcco.EndDate).Exec()
	if err != nil {
		rs.logger.Println(err)
		return err
	}

	return nil
}

func (rs *ReservationRepo) GetReservationsDatesByHostId(host_id string, ctx context.Context) (ReservationsByAccommodation, error) {
	ctx, span := rs.tracer.Start(ctx, "NotificationRepo.GetReservationsDatesByHostId")
	defer span.End()

	scanner := rs.session.Query(`SELECT reservation_id, acco_id, startDate, endDate, host_id, numberPeople, priceByAcoommodation, priceByPeople FROM reservations_by_acco2 WHERE host_id = ?;`,
		host_id).Iter().Scanner() // lista
	var reservations ReservationsByAccommodation
	for scanner.Next() {
		var res ReservationByAccommodation
		err := scanner.Scan(&res.ReservationId, &res.AccoId, &res.StartDate, &res.EndDate, &res.HostId, &res.NumberPeople, &res.PriceByAccommodation, &res.PriceByPeople)
		if err != nil {
			rs.logger.Println("Cant 1", err)
			return nil, err
		}
		reservations = append(reservations, &res)
	}
	if err := scanner.Err(); err != nil {
		rs.logger.Println("Cant 2", err)
		return nil, err
	}
	log.Println(reservations)
	return reservations, nil
}

// RESERVATION DATE FOR ACCO
func (rs *ReservationRepo) GetReservationsDatesByAccommodationId(acco_id string, ctx context.Context) (ReservationDatesByAccomodationId, error) {
	ctx, span := rs.tracer.Start(ctx, "NotificationRepo.GetReservationsDatesByAccommodationId")
	defer span.End()

	scanner := rs.session.Query(`SELECT begin_reservation_date, end_reservation_date
    FROM reservations_dates_by_acco_id
    WHERE accommodation_id = ?;`, // teba videi da li ce trebati isDeleted
		acco_id).Iter().Scanner()

	var dates ReservationDatesByAccomodationId
	for scanner.Next() {
		var res ReservationDate
		err := scanner.Scan(&res.BeginAccomodationDate, &res.EndAccomodationDate)
		if err != nil {
			rs.logger.Println(err)
			return nil, err
		}
		dates = append(dates, &res)
	}
	if err := scanner.Err(); err != nil {
		rs.logger.Println(err)
		return nil, err
	}
	return dates, nil
}

func (rs *ReservationRepo) InsertReservationDateForAccommodation(resDate *ReservationDateByDate) error { // -----------------------
	log.Println("Usli u Insert")

	// overlap, err := rs.CheckOverlap(resDate.AccoId, resDate.BeginAccomodationDate, resDate.EndAccomodationDate)
	// if err != nil {
	// 	return err
	// }

	// if overlap {
	// 	return errors.New("overlap detected: Cannot insert overlapping date range")
	// }

	// id, _ := gocql.RandomUUID()

	// err = rs.session.Query(
	// 	`INSERT INTO reservations_dates_by_acco_id (id, accommodation_id, begin_reservation_date, end_reservation_date)
	// 	VALUES (?, ?, ?, ?);`,
	// 	id, resDate.AccoId, resDate.BeginAccomodationDate, resDate.EndAccomodationDate).Exec()
	// if err != nil {
	// 	rs.logger.Println(err)
	// 	return err
	// }
	// log.Println("Insert prosao")
	return nil
}

// SEARCH - RESERVATION DATES BY START AND END DATE
func (rs *ReservationRepo) GetReservationsDatesByDate(beginReservationDate string, endReservationDate string, ctx context.Context) (ReservationDatesByDateGet, error) {
	ctx, span := rs.tracer.Start(ctx, "NotificationRepo.GetReservationsDatesByDate")
	defer span.End()

	scanner := rs.session.Query(`SELECT accommodation_id FROM reservations_dates_by_date
    WHERE begin_reservation_date = ? AND end_reservation_date = ?`,
		beginReservationDate, endReservationDate).Iter().Scanner()

	var accoIds ReservationDatesByDateGet
	for scanner.Next() {
		var res ReservationDateByDateGet
		err := scanner.Scan(&res.AccoId)
		if err != nil {
			rs.logger.Println(err)
			return nil, err
		}
		accoIds = append(accoIds, &res)
	}
	if err := scanner.Err(); err != nil {
		rs.logger.Println(err)
		return nil, err
	}
	return accoIds, nil
}

func (rs *ReservationRepo) InsertReservationDateByDate(resDate *ReservationDateByDate, ctx context.Context) error {
	ctx, span := rs.tracer.Start(ctx, "NotificationRepo.InsertReservationDateByDate")
	defer span.End()

	reservationId, _ := gocql.RandomUUID()
	err := rs.session.Query(
		`INSERT INTO reservations_dates_by_date (id, accommodation_id, begin_reservation_date, end_reservation_date) 
		VALUES (?, ?, ?, ?);`,
		reservationId, resDate.AccoId, resDate.BeginAccomodationDate, resDate.EndAccomodationDate).Exec()
	if err != nil {
		rs.logger.Println(err)
		return err
	}
	return nil
}

// -------Reservation By User-------//
func (rs *ReservationRepo) GetReservationsByUser(user_id string, ctx context.Context) (ReservationsByUser, error) {
	ctx, span := rs.tracer.Start(ctx, "NotificationRepo.GetReservationsByUser")
	defer span.End()

	scanner := rs.session.Query(`SELECT reservation_id, acco_id, price, 
	begin_reservation_date, numberOfPeople, end_reservation_date
	FROM reservations_by_user WHERE user_id = ?;`,
		user_id).Iter().Scanner()

	var reservations ReservationsByUser
	for scanner.Next() {
		var res UserReservations
		err := scanner.Scan(&res.ReservationId, &res.AccoId, &res.Price,
			&res.StartDate, &res.NumberOfPeople, &res.EndDate)
		if err != nil {
			rs.logger.Println(err)
			return nil, err
		}
		reservations = append(reservations, &res)
	}
	if err := scanner.Err(); err != nil {
		rs.logger.Println(err)
		return nil, err
	}
	return reservations, nil
}

func (rs *ReservationRepo) InsertReservationByUser(resUser *ReservationByUser, ctx context.Context) error {
	ctx, span := rs.tracer.Start(ctx, "NotificationRepo.InsertReservationByUser")
	defer span.End()

	log.Println("Usli u metodu")

	response, err := rs.isDatePassed(resUser.StartDate)
	if err != nil {
		// log.Println("e")
	}

	if response {
		return errors.New("Cant reserve in past")
	}

	overlap, err := rs.CheckOverlap(resUser.AccoId, resUser.StartDate, resUser.EndDate)
	if err != nil {
		return err
	}

	if overlap {
		return errors.New("Dates are already reserved for that accommodation")
	}

	log.Println("Reservation2:", resUser)

	err = rs.session.Query(
		`INSERT INTO reservations_by_user (user_id, reservation_id, acco_id, price, 
			begin_reservation_date, numberOfPeople, end_reservation_date) 
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		resUser.UserId, resUser.ReservationId, resUser.AccoId, 100,
		resUser.StartDate, 2, resUser.EndDate).Exec()
	if err != nil {
		rs.logger.Println(err)
		return err
	}

	err = rs.session.Query(
		`INSERT INTO reservations_dates_by_acco_id (accommodation_id, begin_reservation_date, end_reservation_date) 
		VALUES (?, ?, ?);`,
		resUser.AccoId, resUser.StartDate, resUser.EndDate).Exec()
	if err != nil {
		rs.logger.Println(err)
		return err
	}
	return nil
}

//--------------//

func (rs *ReservationRepo) UpdateReservationByAcco(accoId string, reservationId string, hostId string) error { // nije namesteno
	// err := rs.session.Query(
	// 	`DELETE FROM reservations_by_acco1 where acoo_id = ? and reservation_id = ?`, // delete
	// 	accoId, reservationId).Exec()
	// if err != nil {
	// 	rs.logger.Println(err)
	// 	return err
	// }
	// err = rs.session.Query(
	// 	`DELETE FROM reservations_by_acco2 where host_id = ? and reservation_id = ?`, // delete
	// 	hostId, reservationId).Exec()
	// if err != nil {
	// 	rs.logger.Println(err)
	// 	return err
	// }
	return nil
}

func (rs *ReservationRepo) CheckTable(user_id string, reservation_id string, acco_id string, start_date time.Time, end_date time.Time) (bool, error) {
	log.Println("Start: ", start_date.Format("2006-01-02"))
	log.Println("End: ", end_date.Format("2006-01-02"))

	var count int
	err := rs.session.Query(
		`SELECT COUNT(*) FROM reservation.reservations_by_user
		WHERE user_id = ? 
		AND reservation_id = ?
		AND acco_id = ?
		AND begin_reservation_date = ?
		AND end_reservation_date = ? ALLOW FILTERING`,
		user_id, reservation_id, acco_id, start_date, end_date).Scan(&count)

	if err != nil {
		rs.logger.Println(err)
		return false, err
	}

	log.Println("Count: ", count)

	return count > 0, nil
}

func (rs *ReservationRepo) CheckOverlap(accommodationID string, beginDate, endDate time.Time) (bool, error) {
	var count int
	err := rs.session.Query(
		`SELECT COUNT(*) FROM reservations_by_user
         WHERE acco_id = ? 
         AND begin_reservation_date <= ? AND end_reservation_date >= ? ALLOW FILTERING`,
		accommodationID, endDate, beginDate).Scan(&count)

	if err != nil {
		rs.logger.Println(err)
		return false, err
	}

	log.Println("Count: ", count)

	return count > 0, nil
}
func (rs *ReservationRepo) CheckOverlap1(accommodationID string, beginDate, endDate time.Time) (bool, error) {
	var count int
	err := rs.session.Query(
		`SELECT COUNT(*) FROM reservations_by_acco1
         WHERE acco_id = ? 
         AND startDate <= ? AND endDate >= ? ALLOW FILTERING`,
		accommodationID, endDate, beginDate).Scan(&count)

	if err != nil {
		rs.logger.Println(err)
		return false, err
	}

	log.Println("Count: ", count)

	return count > 0, nil
}

func (rs *ReservationRepo) UpdateReservationByUser(reservationByUser *ReservationByUser, ctx context.Context) error {
	ctx, span := rs.tracer.Start(ctx, "NotificationRepo.UpdateReservationByUser")
	defer span.End()

	overlap, err := rs.CheckTable(reservationByUser.UserId, reservationByUser.ReservationId, reservationByUser.AccoId, reservationByUser.StartDate, reservationByUser.EndDate)
	if err != nil {
		return err
	}

	log.Println("Overlap:", overlap)

	if !overlap {
		return errors.New("Cant find reservation")
	}

	passed, err := rs.isDatePassed(reservationByUser.StartDate)
	if err != nil {
		log.Println("Error:", err)
		return err
	}

	if passed {
		return errors.New("Reservation cant be canceled")
	}

	// treba da se postavi validacija da li je datum prosao ako jeste onda ne moze da otkaze rezervaciju

	err = rs.session.Query(
		`DELETE from reservations_by_user where user_id = ? and reservation_id = ? and acco_id = ? and begin_reservation_date = ? and end_reservation_date = ? and price = ?`,
		reservationByUser.UserId, reservationByUser.ReservationId, reservationByUser.AccoId, reservationByUser.StartDate, reservationByUser.EndDate, reservationByUser.Price).Exec()

	if err != nil {
		rs.logger.Println(err)
		return err
	}

	err = rs.session.Query(
		`DELETE from reservations_dates_by_acco_id where accommodation_id = ? and begin_reservation_date = ? and end_reservation_date = ?;`,
		reservationByUser.AccoId, reservationByUser.StartDate, reservationByUser.EndDate).Exec()

	if err != nil {
		rs.logger.Println(err)
		return err
	}

	return nil
}

func (rs *ReservationRepo) GetDistinctIds(idColumnName string, tableName string) ([]string, error) {
	scanner := rs.session.Query(
		fmt.Sprintf(`SELECT DISTINCT %s FROM %s`, idColumnName, tableName)).
		Iter().Scanner()
	var ids []string
	for scanner.Next() {
		var id string
		err := scanner.Scan(&id)
		if err != nil {
			rs.logger.Println(err)
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := scanner.Err(); err != nil {
		rs.logger.Println(err)
		return nil, err
	}
	return ids, nil
}

func (rs *ReservationRepo) isDatePassed(dateStr time.Time) (bool, error) {
	currentDate := time.Now()
	return dateStr.Before(currentDate), nil
}
