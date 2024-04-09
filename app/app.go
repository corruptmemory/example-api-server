package app

import (
	"io"
	"slices"
	"sync"

	"example-api-server/appinterface"
)

type appCommandTag int

const (
	addContact appCommandTag = iota
	getContacts
	contactDetails
	deleteContact
	updateContact
)

type appCommand struct {
	tag       appCommandTag
	inContact appinterface.Contact
	result    chan any
}

type app struct {
	commands chan appCommand
	wg       *sync.WaitGroup
}

func (a *app) AddContact(firstName string, lastName string, email string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()
	a.commands <- appCommand{
		tag: addContact,
		inContact: appinterface.Contact{
			FirstName: firstName,
			LastName:  lastName,
			Email:     email,
		},
	}
	return
}

func (a *app) GetContacts() (result []appinterface.Contact, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()
	r := make(chan any, 1)
	a.commands <- appCommand{
		tag:    getContacts,
		result: r,
	}
	return (<-r).([]appinterface.Contact), nil
}

func (a *app) ContactDetails(id int) (result appinterface.Contact, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()
	r := make(chan any, 1)
	a.commands <- appCommand{
		tag: contactDetails,
		inContact: appinterface.Contact{
			ID: id,
		},
		result: r,
	}
	result, ok := (<-r).(appinterface.Contact)
	if !ok {
		return result, io.EOF
	}
	return result, nil
}

func (a *app) DeleteContact(id int) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()

	a.commands <- appCommand{
		tag: deleteContact,
		inContact: appinterface.Contact{
			ID: id,
		},
	}
	return
}

func (a *app) UpdateContact(id int, firstName string, lastName string, email string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()
	a.commands <- appCommand{
		tag: updateContact,
		inContact: appinterface.Contact{
			ID:        id,
			FirstName: firstName,
			LastName:  lastName,
			Email:     email,
		},
	}
	return nil
}

func (a *app) Stop() {
	close(a.commands)
}

func (a *app) Wait() {
	a.wg.Wait()
}

func NewApp(queueSize int) appinterface.App {
	if queueSize < 10 {
		queueSize = 10
	}
	wg := &sync.WaitGroup{}
	r := &app{
		commands: make(chan appCommand, queueSize),
		wg:       wg,
	}
	go r.run()
	return r
}

func (a *app) run() {
	defer func() {
		a.wg.Done()
	}()
	currentID := 0
	var contacts []appinterface.Contact
	sortContacts := func() {
		slices.SortFunc(contacts, func(a, b appinterface.Contact) int {
			switch {
			case a.FirstName < b.FirstName:
				return -1
			case a.FirstName > b.FirstName:
				return 1
			case a.LastName < b.LastName:
				return -1
			case a.LastName > b.LastName:
				return 1
			case a.Email < b.Email:
				return -1
			case a.Email > b.Email:
				return 1
			}
			return 0
		})
	}
	findIndexByContent := func(contact appinterface.Contact) (int, bool) {
		return slices.BinarySearchFunc(contacts, contact, func(a, b appinterface.Contact) int {
			switch {
			case a.FirstName < b.FirstName:
				return -1
			case a.FirstName > b.FirstName:
				return 1
			case a.LastName < b.LastName:
				return -1
			case a.LastName > b.LastName:
				return 1
			case a.Email < b.Email:
				return -1
			case a.Email > b.Email:
				return 1
			}
			return 0
		})
	}
	findIndexByID := func(id int) int {
		return slices.IndexFunc(contacts, func(a appinterface.Contact) bool {
			return a.ID == id
		})
	}

	ac := func(firstName string, lastName string, email string) {
		contact := appinterface.Contact{
			FirstName: firstName,
			LastName:  lastName,
			Email:     email,
		}
		_, ok := findIndexByContent(contact)
		if !ok {
			currentID++
			contact.ID = currentID
			contacts = append(contacts, contact)
			sortContacts()
		}
	}

	uc := func(id int, firstName string, lastName string, email string) {
		idx := findIndexByID(id)
		if idx >= 0 {
			contacts[idx].FirstName = firstName
			contacts[idx].LastName = lastName
			contacts[idx].Email = email
			sortContacts()
		}
	}

	for cmd := range a.commands {
		switch cmd.tag {
		case addContact:
			ac(cmd.inContact.FirstName, cmd.inContact.LastName, cmd.inContact.Email)
		case getContacts:
			cpy := make([]appinterface.Contact, len(contacts))
			copy(cpy, contacts)
			cmd.result <- cpy
			close(cmd.result)
		case contactDetails:
			idx := findIndexByID(cmd.inContact.ID)
			if idx >= 0 {
				cmd.result <- contacts[idx]
			}
			close(cmd.result)
		case deleteContact:
			idx := findIndexByID(cmd.inContact.ID)
			if idx >= 0 {
				slices.Delete(contacts, idx, idx+1)
				contacts = contacts[:len(contacts)-1]
			}
		case updateContact:
			uc(cmd.inContact.ID, cmd.inContact.FirstName, cmd.inContact.LastName, cmd.inContact.Email)
		}
	}
}
