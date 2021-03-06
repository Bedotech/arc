/*
 * Arc - Copyleft of Simone 'evilsocket' Margaritelli.
 * evilsocket at protonmail dot com
 * https://www.evilsocket.net/
 *
 * See LICENSE.
 */
package events

import (
	"crypto/tls"
	"fmt"
	"github.com/evilsocket/arc/arcd/config"
	"github.com/evilsocket/arc/arcd/log"
	"github.com/evilsocket/arc/arcd/pgp"
	"github.com/evilsocket/arc/arcd/utils"
	"gopkg.in/gomail.v2"
	"sync"
)

var (
	lock    = &sync.Mutex{}
	Pool    = make([]Event, 0)
	pgpConf = &config.Conf.Scheduler.Reports.PGP
)

func Setup() error {
	reports := config.Conf.Scheduler.Reports
	if config.Conf.Scheduler.Enabled && reports.Enabled && pgpConf.Enabled {
		if err := pgp.Setup(pgpConf); err != nil {
			return err
		}
	}
	return nil
}

func Report(event Event) {
	repotype := "plaintext"
	if pgpConf.Enabled {
		repotype = "PGP encrypted"
	}

	log.Infof("Reporting %s event '%s' to %s ...", repotype, event.Title, config.Conf.Scheduler.Reports.To)

	smtp := config.Conf.Scheduler.Reports.SMTP
	d := gomail.NewDialer(smtp.Address, smtp.Port, smtp.Username, smtp.Password)
	d.TLSConfig = &tls.Config{InsecureSkipVerify: true}

	m := gomail.NewMessage()
	m.SetHeader("From", fmt.Sprintf("Arc Reporting System <%s>", smtp.Username))
	m.SetHeader("To", config.Conf.Scheduler.Reports.To)
	m.SetHeader("Subject", event.Title)

	var err error
	ctype := "text/html"
	body := event.Description
	if pgpConf.Enabled {
		ctype = "text/plain"
		if err, body = pgp.Encrypt(body); err != nil {
			log.Errorf("Could not PGP encrypt the message: %s.", err)
		}
	}

	m.SetBody(ctype, body)

	if err := d.DialAndSend(m); err != nil {
		log.Errorf("Error: %s.", err)
	} else {
		log.Infof("Reported.")
	}
}

func Add(event Event) {
	lock.Lock()
	defer lock.Unlock()
	Pool = append([]Event{event}, Pool...)
	log.Debugf("New event added (Pool size is %d): %s.", len(Pool), event)

	if config.Conf.Scheduler.Reports.Enabled && utils.InSlice(event.Name, config.Conf.Scheduler.Reports.Filter) == true {
		go Report(event)
	}
}

func Clear() {
	lock.Lock()
	defer lock.Unlock()
	Pool = make([]Event, 0)
	log.Debugf("Events Pool has been cleared.")
}

func AddNew(name, title, description string) Event {
	event := New(name, title, description)
	Add(event)
	return event
}
