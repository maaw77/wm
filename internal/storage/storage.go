package storage

import (
	"errors"
	"sync"
)

var (
	ErrNotExist = errors.New("it doesn't exist")
)

type LinkStatus struct {
	URLs   []string
	Status string
}

// Storage определяет интерфейс для хранения и управления задачами проверки ссылок.
type Storage interface {
	// Add создает новую задачу с указанными ссылками и возвращает присвоенный ID.
	Add(links []string) uint64
	// Get возвращает задачу по указанному ID. Возвращает ErrNotExist, если задача не найдена.
	Get(id uint64) (LinkStatus, error)
	// GetAll итерируется по всем задачам и вызывает функцию fn для каждой пары (id, LinkStatus).
	GetAll(fn func(id uint64, links LinkStatus) bool)
}

type StorageLinkStatus struct {
	mu     sync.RWMutex
	nextID uint64
	Links  []LinkStatus
}

// Add создает новую задачу с указанными ссылками и возвращает присвоенный ID.
func (s *StorageLinkStatus) Add(links []string) uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextID++
	id := s.nextID

	task := LinkStatus{
		URLs:   links,
		Status: "", // статус будет установлен позже при обработке
	}

	s.Links = append(s.Links, task)
	return id
}

// Get возвращает задачу по указанному ID. Возвращает ErrNotExist, если задача не найдена.
func (s *StorageLinkStatus) Get(id uint64) (LinkStatus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if id == 0 || id > uint64(len(s.Links)) {
		return LinkStatus{}, ErrNotExist
	}

	// ID начинается с 1, индекс массива с 0
	index := id - 1
	return s.Links[index], nil
}

// GetAll итерируется по всем задачам и вызывает функцию fn для каждой пары (id, LinkStatus).
func (s *StorageLinkStatus) GetAll(fn func(id uint64, links LinkStatus) bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for i, status := range s.Links {
		id := uint64(i + 1) // ID начинается с 1
		if !fn(id, status) {
			break
		}
	}
}

// NewStorage создает и возвращает новый экземпляр хранилища задач.
func NewStorage() Storage {
	return &StorageLinkStatus{
		nextID: 0,
		Links:  make([]LinkStatus, 0),
	}
}
