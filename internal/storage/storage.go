package storage

import (
	"encoding/json"
	"errors"
	"os"
	"sync"
)

var (
	ErrNotExist     = errors.New("it doesn't exist")
	ErrEmptyInpData = errors.New("input data is empty or nil")
)

// LinkStatus описывает "задачу": только результаты по каждой ссылке.
type LinkStatus struct {
	Results map[string]string `json:"results"`
}

type Storage interface {
	Add(links []string) (uint64, error)
	Get(id uint64) (LinkStatus, error)
	GetAll(fn func(id uint64, links LinkStatus) bool)
	UpdateResults(id uint64, results map[string]string) error
	Dump(filename string) error
	Upload(filename string) error
}

type StorageLinkStatus struct {
	mu     sync.RWMutex
	nextID uint64
	links  []LinkStatus
}

func NewStorage() Storage {
	return &StorageLinkStatus{
		nextID: 0,
		links:  make([]LinkStatus, 0),
	}
}

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

func (s *StorageLinkStatus) Get(id uint64) (LinkStatus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if id == 0 || id > uint64(len(s.links)) {
		return LinkStatus{}, ErrNotExist
	}

	index := id - 1
	return s.links[index], nil
}

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
