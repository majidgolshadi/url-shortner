package id

type Generator interface {
	GetLastID() uint
	NewID() uint
}
