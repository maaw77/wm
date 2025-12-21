// Package storage реализует in-memory хранилище результатов проверки ссылок.
//
// Хранилище поддерживает:
// - добавление новой записи со списком ссылок (Add);
// - получение записи по ID (Get);
// - итерацию по всем записям (GetAll);
// - обновление результатов проверки (UpdateResults);
// - сохранение/восстановление состояния в JSON-файл (Dump/Upload).
package storage

import (
	"encoding/json"
	"errors"
	"os"
	"sync"
)

var (
	// ErrNotExist возвращается при обращении к несуществующему ID записи.
	ErrNotExist = errors.New("it doesn't exist")
	// ErrEmptyInpData возвращается при попытке создать ззапись с пустым или nil списком ссылок.
	ErrEmptyInpData = errors.New("input data is empty or nil")
)

// LinkStatus описывает "запись": только результаты по каждой ссылке.
type LinkStatus struct {
	Results map[string]string `json:"results"`
}

// Storage описывает интерфейс in-memory хранилища записей.
type Storage interface {
	// Add создает новую запись и возвращает её ID (начиная с 1).
	Add(links []string) (uint64, error)

	// Get возвращает запись по ID.
	Get(id uint64) (LinkStatus, error)

	// GetAll — итератор по всем записям хранилища.
	GetAll(fn func(id uint64, links LinkStatus) bool)

	// UpdateResults обновляет статусы ссылок в записи по ID.
	// Обновляются только те URL, которые уже существуют в записи.
	UpdateResults(id uint64, results map[string]string) error

	// Dump сохраняет текущее состояние в JSON-файл.
	Dump(filename string) error

	// Upload загружает состояние из JSON-файла и восстанавливает записи в памяти.
	Upload(filename string) error
}

// StorageLinkStatus — потокобезопасная реализация Storage на базе []LinkStatus.
type StorageLinkStatus struct {
	mu     sync.RWMutex
	nextID uint64
	links  []LinkStatus
}

// NewStorage создает пустое in-memory хранилище.
func NewStorage() Storage {
	return &StorageLinkStatus{
		nextID: 0,
		links:  make([]LinkStatus, 0),
	}
}

// Add добавляет новую запись со списком ссылок и возвращает её ID.
func (s *StorageLinkStatus) Add(links []string) (uint64, error) {
	if len(links) == 0 {
		return 0, ErrEmptyInpData
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextID++
	id := s.nextID

	results := make(map[string]string, len(links))
	for _, u := range links {
		results[u] = ""
	}

	task := LinkStatus{
		Results: results,
	}

	s.links = append(s.links, task)

	return id, nil
}

// Get возвращает запись по ID.
func (s *StorageLinkStatus) Get(id uint64) (LinkStatus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if id == 0 || id > uint64(len(s.links)) {
		return LinkStatus{}, ErrNotExist
	}

	index := id - 1
	return s.links[index], nil
}

// GetAll итерируется по всем запись.
func (s *StorageLinkStatus) GetAll(fn func(id uint64, links LinkStatus) bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for i, status := range s.links {
		id := uint64(i + 1)
		if !fn(id, status) {
			break
		}
	}
}

// UpdateResults обновляет результаты по ссылкам,  указанным ID записи.
func (s *StorageLinkStatus) UpdateResults(id uint64, results map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if id == 0 || id > uint64(len(s.links)) {
		return ErrNotExist
	}

	index := id - 1
	task := s.links[index]

	if task.Results == nil {
		task.Results = make(map[string]string, len(results))
	}

	for url, status := range results {
		if _, ok := task.Results[url]; ok {
			task.Results[url] = status
		}
	}

	s.links[index] = task
	return nil
}

type taskWithID struct {
	ID      uint64            `json:"id"`
	Results map[string]string `json:"results"`
}

type storageData struct {
	NextID      uint64       `json:"nextid"`
	LinkEntries []taskWithID `json:"linkentries"`
}

// Dump сохраняет состояние хранилища в JSON-файл.
func (s *StorageLinkStatus) Dump(filename string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data := storageData{
		NextID:      s.nextID,
		LinkEntries: make([]taskWithID, 0, len(s.links)),
	}

	for i, linkStatus := range s.links {
		id := uint64(i + 1)
		data.LinkEntries = append(data.LinkEntries, taskWithID{
			ID:      id,
			Results: linkStatus.Results,
		})
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, jsonData, 0o644)
}

// Upload загружает состояние хранилища из JSON-файла.
func (s *StorageLinkStatus) Upload(filename string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	jsonData, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	var data storageData
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return err
	}

	s.nextID = data.NextID
	s.links = make([]LinkStatus, 0, len(data.LinkEntries))

	for _, task := range data.LinkEntries {
		res := task.Results
		if res == nil {
			res = map[string]string{}
		}

		s.links = append(s.links, LinkStatus{
			Results: res,
		})
	}

	return nil
}
