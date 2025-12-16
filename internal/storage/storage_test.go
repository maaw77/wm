package storage

import (
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"testing"
)

// getStorageLinkStatus выполняет type assertion для доступа к внутренним полям хранилища.
func getStorageLinkStatus(s Storage) *StorageLinkStatus {
	return s.(*StorageLinkStatus)
}

// TestAdd проверяет базовую функциональность добавления задач.
func TestAdd(t *testing.T) {
	s := NewStorage()
	storage := getStorageLinkStatus(s)

	tests := []struct {
		name       string
		links      []string
		expectedID uint64
		wantError  bool
	}{
		{
			name:       "добавление задачи 1",
			links:      []string{"google.com", "yandex.ru"},
			expectedID: 1,
			wantError:  false,
		},
		{
			name:       "добавление задачи 2",
			links:      []string{"github.com"},
			expectedID: 2,
			wantError:  false,
		},
		{
			name:      "пустой список задач",
			links:     []string{},
			wantError: true,
		},
		{
			name:      "с nil",
			links:     nil,
			wantError: true,
		},
	}

	successCount := 0
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := s.Add(tt.links)
			if tt.wantError {
				if err == nil {
					t.Error("Add() error = nil, want ErrEmptyInpData")
				} else if !errors.Is(err, ErrEmptyInpData) {
					t.Errorf("Add() error = %v, want ErrEmptyInpData", err)
				}
				return
			}

			if err != nil {
				t.Errorf("Add() error = %v, want nil", err)
			}
			if id != tt.expectedID {
				t.Errorf("Expected ID %d, got %d", tt.expectedID, id)
			}

			// Проверяем, что задача сохранилась
			index := id - 1
			if index >= uint64(len(storage.links)) {
				t.Fatalf("Task index %d out of range (len=%d)", index, len(storage.links))
			}

			task := storage.links[index]
			if !reflect.DeepEqual(task.URLs, tt.links) {
				t.Errorf("Expected URLs %v, got %v", tt.links, task.URLs)
			}

			// Results должен быть инициализирован и содержать все URL с пустыми статусами.
			if task.Results == nil {
				t.Fatal("Expected non-nil Results map")
			}
			if len(task.Results) != len(tt.links) {
				t.Fatalf("Expected Results size %d, got %d", len(tt.links), len(task.Results))
			}
			for _, u := range tt.links {
				v, ok := task.Results[u]
				if !ok {
					t.Fatalf("Expected Results to contain key %q", u)
				}
				if v != "" {
					t.Fatalf("Expected empty status for %q, got %q", u, v)
				}
			}

			successCount++
		})
	}

	// Проверяем общее количество задач
	if len(storage.links) != successCount {
		t.Errorf("Expected %d tasks, got %d", successCount, len(storage.links))
	}
}

// TestGet проверяет получение задач по ID.
func TestGet(t *testing.T) {
	s := NewStorage()

	links1 := []string{"google.com"}
	links2 := []string{"yandex.ru", "github.com"}
	links3 := []string{"example.com"}

	id1, err := s.Add(links1)
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	id2, err := s.Add(links2)
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	id3, err := s.Add(links3)
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	tests := []struct {
		name      string
		id        uint64
		wantFound bool
		wantURLs  []string
	}{
		{"получение первой задачи", id1, true, links1},
		{"получение второй задачи", id2, true, links2},
		{"получение третьей задачи", id3, true, links3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task, err := s.Get(tt.id)
			if tt.wantFound {
				if err != nil {
					t.Errorf("Get() error = %v, want nil", err)
				}
			} else {
				if err == nil {
					t.Error("Get() error = nil, want ErrNotExist")
				} else if !errors.Is(err, ErrNotExist) {
					t.Errorf("Get() error = %v, want ErrNotExist", err)
				}
			}

			if !reflect.DeepEqual(task.URLs, tt.wantURLs) {
				t.Errorf("Get() URLs = %v, want %v", task.URLs, tt.wantURLs)
			}

			if task.Results == nil {
				t.Error("Expected non-nil Results map")
			}
			for _, u := range tt.wantURLs {
				if _, ok := task.Results[u]; !ok {
					t.Errorf("Expected Results to contain key %q", u)
				}
			}
		})
	}
}

// TestGetInvalidID проверяет получение задач с невалидными ID.
func TestGetInvalidID(t *testing.T) {
	s := NewStorage()

	_, err := s.Add([]string{"google.com"})
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	tests := []struct {
		name      string
		id        uint64
		wantFound bool
		wantEmpty bool
	}{
		{"ID == 0", 0, false, true},
		{"ID > кол-ва задач", 2, false, true},
		{"валидный ID", 1, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task, err := s.Get(tt.id)
			if tt.wantFound {
				if err != nil {
					t.Errorf("Get() error = %v, want nil", err)
				}
			} else {
				if err == nil {
					t.Error("Get() error = nil, want ErrNotExist")
				} else if !errors.Is(err, ErrNotExist) {
					t.Errorf("Get() error = %v, want ErrNotExist", err)
				}
			}

			if tt.wantEmpty {
				if task.URLs != nil || task.Results != nil {
					t.Errorf("Expected empty task, got %+v", task)
				}
			} else {
				if task.URLs == nil {
					t.Error("Expected non-empty task URLs")
				}
				if task.Results == nil {
					t.Error("Expected non-nil Results map")
				}
			}
		})
	}
}

// TestGetAll проверяет итерацию по всем задачам.
func TestGetAll(t *testing.T) {
	s := NewStorage()

	links1 := []string{"google.com"}
	links2 := []string{"yandex.ru"}
	links3 := []string{"github.com"}

	id1, err := s.Add(links1)
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	id2, err := s.Add(links2)
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	id3, err := s.Add(links3)
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	t.Run("итерация по всем задачам", func(t *testing.T) {
		collected := make(map[uint64]LinkStatus)
		s.GetAll(func(id uint64, links LinkStatus) bool {
			collected[id] = links
			return true
		})

		if len(collected) != 3 {
			t.Errorf("Expected 3 tasks, got %d", len(collected))
		}

		tests := []struct {
			name     string
			id       uint64
			wantURLs []string
		}{
			{"задача 1", id1, links1},
			{"задача 2", id2, links2},
			{"задача 3", id3, links3},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				task, ok := collected[tt.id]
				if !ok {
					t.Fatalf("Task with ID %d not found in collected", tt.id)
				}
				if !reflect.DeepEqual(task.URLs, tt.wantURLs) {
					t.Errorf("Expected URLs %v for ID %d, got %v", tt.wantURLs, tt.id, task.URLs)
				}
				if task.Results == nil {
					t.Fatalf("Expected non-nil Results for ID %d", tt.id)
				}
			})
		}
	})

	t.Run("досрочное прекращение итерации", func(t *testing.T) {
		count := 0
		s.GetAll(func(id uint64, links LinkStatus) bool {
			count++
			return count < 2 // останавливаемся после второй задачи
		})

		if count != 2 {
			t.Errorf("Expected 2 iterations, got %d", count)
		}
	})

	t.Run("итерация по пустому хранилищу", func(t *testing.T) {
		emptyStorage := NewStorage()
		count := 0
		emptyStorage.GetAll(func(id uint64, links LinkStatus) bool {
			count++
			return true
		})

		if count != 0 {
			t.Errorf("Expected 0 iterations, got %d", count)
		}
	})
}

// TestDump проверяет сохранение задач в файл.
func TestDump(t *testing.T) {
	s := NewStorage()
	storage := getStorageLinkStatus(s)

	links1 := []string{"google.com", "yandex.ru"}
	links2 := []string{"github.com"}

	id1, err := s.Add(links1)
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	_, err = s.Add(links2)
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	// Устанавливаем статусы для первой задачи.
	storage.links[id1-1].Results["google.com"] = "available"
	storage.links[id1-1].Results["yandex.ru"] = "not available"

	tmpFile, err := os.CreateTemp("", "test_dump_*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	if err := s.Dump(tmpFile.Name()); err != nil {
		t.Fatalf("Dump() error = %v, want nil", err)
	}

	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read dumped file: %v", err)
	}

	var savedData storageData
	if err := json.Unmarshal(data, &savedData); err != nil {
		t.Fatalf("Failed to unmarshal dumped data: %v", err)
	}

	if savedData.NextID != 2 {
		t.Errorf("Expected NextID 2, got %d", savedData.NextID)
	}
	if len(savedData.LinkEntries) != 2 {
		t.Fatalf("Expected 2 tasks, got %d", len(savedData.LinkEntries))
	}

	// Проверяем первую задачу.
	if savedData.LinkEntries[0].ID != 1 {
		t.Errorf("Expected ID 1, got %d", savedData.LinkEntries[0].ID)
	}
	if !reflect.DeepEqual(savedData.LinkEntries[0].URLs, links1) {
		t.Errorf("Expected URLs %v, got %v", links1, savedData.LinkEntries[0].URLs)
	}
	if savedData.LinkEntries[0].Results["google.com"] != "available" {
		t.Errorf("Expected google.com status 'available', got %q", savedData.LinkEntries[0].Results["google.com"])
	}
	if savedData.LinkEntries[0].Results["yandex.ru"] != "not available" {
		t.Errorf("Expected yandex.ru status 'not available', got %q", savedData.LinkEntries[0].Results["yandex.ru"])
	}

	// Проверяем вторую задачу.
	if savedData.LinkEntries[1].ID != 2 {
		t.Errorf("Expected ID 2, got %d", savedData.LinkEntries[1].ID)
	}
	if !reflect.DeepEqual(savedData.LinkEntries[1].URLs, links2) {
		t.Errorf("Expected URLs %v, got %v", links2, savedData.LinkEntries[1].URLs)
	}
	if savedData.LinkEntries[1].Results == nil {
		t.Fatalf("Expected non-nil Results for task 2")
	}
	if savedData.LinkEntries[1].Results["github.com"] != "" {
		t.Errorf("Expected github.com empty status, got %q", savedData.LinkEntries[1].Results["github.com"])
	}
}

// TestUpload проверяет загрузку задач из файла.
func TestUpload(t *testing.T) {
	testData := storageData{
		NextID: 3,
		LinkEntries: []taskWithID{
			{
				ID:   1,
				URLs: []string{"google.com", "yandex.ru"},
				Results: map[string]string{
					"google.com": "available",
					"yandex.ru":  "not available",
				},
			},
			{
				ID:   2,
				URLs: []string{"github.com"},
				Results: map[string]string{
					"github.com": "",
				},
			},
		},
	}

	tmpFile, err := os.CreateTemp("", "test_upload_*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	jsonData, err := json.MarshalIndent(testData, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal test data: %v", err)
	}

	if err := os.WriteFile(tmpFile.Name(), jsonData, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	s := NewStorage()
	storage := getStorageLinkStatus(s)

	if err := s.Upload(tmpFile.Name()); err != nil {
		t.Fatalf("Upload() error = %v, want nil", err)
	}

	if storage.nextID != 3 {
		t.Errorf("Expected nextID 3, got %d", storage.nextID)
	}
	if len(storage.links) != 2 {
		t.Fatalf("Expected 2 tasks, got %d", len(storage.links))
	}

	// Проверяем первую задачу.
	if !reflect.DeepEqual(storage.links[0].URLs, testData.LinkEntries[0].URLs) {
		t.Errorf("Expected URLs %v, got %v", testData.LinkEntries[0].URLs, storage.links[0].URLs)
	}
	if storage.links[0].Results["google.com"] != "available" {
		t.Errorf("Expected google.com 'available', got %q", storage.links[0].Results["google.com"])
	}

	// Проверяем, что можно получить задачи по ID.
	task1, err := s.Get(1)
	if err != nil {
		t.Errorf("Get(1) error = %v, want nil", err)
	}
	if !reflect.DeepEqual(task1.URLs, testData.LinkEntries[0].URLs) {
		t.Errorf("Get(1) URLs = %v, want %v", task1.URLs, testData.LinkEntries[0].URLs)
	}
	if task1.Results["yandex.ru"] != "not available" {
		t.Errorf("Get(1) yandex.ru = %q, want %q", task1.Results["yandex.ru"], "not available")
	}
}

// TestUploadInvalidFile проверяет обработку ошибок при загрузке из несуществующего файла.
func TestUploadInvalidFile(t *testing.T) {
	s := NewStorage()

	err := s.Upload("nonexistent_file.json")
	if err == nil {
		t.Error("Upload() error = nil, want error")
	}
}

// TestDumpAndUpload проверяет полный цикл сохранения и загрузки.
func TestDumpAndUpload(t *testing.T) {
	s1 := NewStorage()
	links1 := []string{"google.com", "yandex.ru"}
	links2 := []string{"github.com"}

	id1, err := s1.Add(links1)
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	id2, err := s1.Add(links2)
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	storage1 := getStorageLinkStatus(s1)
	storage1.links[id1-1].Results["google.com"] = "available"
	storage1.links[id1-1].Results["yandex.ru"] = "not available"
	storage1.links[id2-1].Results["github.com"] = "not available"

	tmpFile, err := os.CreateTemp("", "test_roundtrip_*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	if err := s1.Dump(tmpFile.Name()); err != nil {
		t.Fatalf("Dump() error = %v, want nil", err)
	}

	s2 := NewStorage()
	if err := s2.Upload(tmpFile.Name()); err != nil {
		t.Fatalf("Upload() error = %v, want nil", err)
	}

	task1, err := s2.Get(1)
	if err != nil {
		t.Errorf("Get(1) error = %v, want nil", err)
	}
	if !reflect.DeepEqual(task1.URLs, links1) {
		t.Errorf("Get(1) URLs = %v, want %v", task1.URLs, links1)
	}
	if task1.Results["google.com"] != "available" {
		t.Errorf("Get(1) google.com = %q, want %q", task1.Results["google.com"], "available")
	}

	task2, err := s2.Get(2)
	if err != nil {
		t.Errorf("Get(2) error = %v, want nil", err)
	}
	if !reflect.DeepEqual(task2.URLs, links2) {
		t.Errorf("Get(2) URLs = %v, want %v", task2.URLs, links2)
	}
	if task2.Results["github.com"] != "not available" {
		t.Errorf("Get(2) github.com = %q, want %q", task2.Results["github.com"], "not available")
	}

	storage2 := getStorageLinkStatus(s2)
	if storage2.nextID != 2 {
		t.Errorf("Expected nextID 2, got %d", storage2.nextID)
	}
}
