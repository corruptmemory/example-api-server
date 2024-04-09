package appinterface

type Contact struct {
	ID        int    `json:"id"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
}

type App interface {
	AddContact(firstName string, lastName string, email string) error
	GetContacts() ([]Contact, error)
	ContactDetails(id int) (Contact, error)
	DeleteContact(id int) error
	UpdateContact(id int, firstName string, lastName string, email string) error
	Stop()
	Wait()
}
