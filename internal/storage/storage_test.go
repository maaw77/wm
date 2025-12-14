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
	}{
		{
			name:       "добавление задачи 1",
			links:      []string{"google.com", "yandex.ru"},
			expectedID: 1,
		},
		{
			name:       "добавление задачи 2",
			links:      []string{"github.com"},
			expectedID: 2,
		},
		{
			name:       "пустой список задач",
			links:      []string{},
			expectedID: 3,
		},
		{
			name:       "с nil",
			links:      nil,
			expectedID: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := s.Add(tt.links)
			if id != tt.expectedID {
				t.Errorf("Expected ID %d, got %d", tt.expectedID, id)
			}

			// Проверяем, что задача сохранилась
			index := id - 1
			if index >= uint64(len(storage.Links)) {
				t.Fatalf("Task index %d out of range (len=%d)", index, len(storage.Links))
			}

			task := storage.Links[index]
			if !reflect.DeepEqual(task.URLs, tt.links) {
				t.Errorf("Expected URLs %v, got %v", tt.links, task.URLs)
			}
			if task.Status != "" {
				t.Errorf("Expected empty status, got %s", task.Status)
			}
		})
	}

	// Проверяем общее количество задач
	if len(storage.Links) != len(tests) {
		t.Errorf("Expected %d tasks, got %d", len(tests), len(storage.Links))
	}
}

// TestGet проверяет получение задач по ID.
func TestGet(t *testing.T) {
	s := NewStorage()

	// Подготавливаем данные
	links1 := []string{"google.com"}
	links2 := []string{"yandex.ru", "github.com"}
	links3 := []string{"example.com"}

	id1 := s.Add(links1)
	id2 := s.Add(links2)
	id3 := s.Add(links3)

	tests := []struct {
		name      string
		id        uint64
		wantFound bool
		wantURLs  []string
	}{
		{
			name:      "получение первой задачи",
			id:        id1,
			wantFound: true,
			wantURLs:  links1,
		},
		{
			name:      "получение второй задачи",
			id:        id2,
			wantFound: true,
			wantURLs:  links2,
		},
		{
			name:      "получение третьей задачи",
			id:        id3,
			wantFound: true,
			wantURLs:  links3,
		},
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
		})
	}
}

// TestGetInvalidID проверяет получение задач с невалидными ID.
func TestGetInvalidID(t *testing.T) {
	s := NewStorage()

	// Добавляем одну задачу для проверки
	s.Add([]string{"google.com"})

	tests := []struct {
		name      string
		id        uint64
		wantFound bool
		wantEmpty bool
	}{
		{
			name:      "ID == 0",
			id:        0,
			wantFound: false,
			wantEmpty: true,
		},
		{
			name:      "ID > кол-ва задач",
			id:        2,
			wantFound: false,
			wantEmpty: true,
		},

		{
			name:      "валидный ID",
			id:        1,
			wantFound: true,
			wantEmpty: false,
		},
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
				if task.URLs != nil || task.Status != "" {
					t.Errorf("Expected empty task, got %+v", task)
				}
			} else {
				if task.URLs == nil {
					t.Error("Expected non-empty task URLs")
				}
			}
		})
	}
}

// TestGetAll проверяет итерацию по всем задачам.
func TestGetAll(t *testing.T) {
	s := NewStorage()

	// Подготавливаем данные
	links1 := []string{"google.com"}
	links2 := []string{"yandex.ru"}
	links3 := []string{"github.com"}

	id1 := s.Add(links1)
	id2 := s.Add(links2)
	id3 := s.Add(links3)

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

	// Подготавливаем данные
	links1 := []string{"google.com", "yandex.ru"}
	links2 := []string{"github.com"}

	id1 := s.Add(links1)
	s.Add(links2)

	// Устанавливаем статус для первой задачи
	storage.Links[id1-1].Status = "available"

	// Создаем временный файл
	tmpFile, err := os.CreateTemp("", "test_dump_*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	// Сохраняем данные
	if err := s.Dump(tmpFile.Name()); err != nil {
		t.Fatalf("Dump() error = %v, want nil", err)
	}

	// Проверяем содержимое файла
	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read dumped file: %v", err)
	}

	var savedData storageData
	if err := json.Unmarshal(data, &savedData); err != nil {
		t.Fatalf("Failed to unmarshal dumped data: %v", err)
	}

	// Проверяем nextID
	if savedData.NextID != 2 {
		t.Errorf("Expected NextID 2, got %d", savedData.NextID)
	}

	// Проверяем количество задач
	if len(savedData.LinkEntries) != 2 {
		t.Fatalf("Expected 2 tasks, got %d", len(savedData.LinkEntries))
	}

	// Проверяем первую задачу
	if savedData.LinkEntries[0].ID != 1 {
		t.Errorf("Expected ID 1, got %d", savedData.LinkEntries[0].ID)
	}
	if !reflect.DeepEqual(savedData.LinkEntries[0].URLs, links1) {
		t.Errorf("Expected URLs %v, got %v", links1, savedData.LinkEntries[0].URLs)
	}
	if savedData.LinkEntries[0].Status != "available" {
		t.Errorf("Expected status 'available', got %s", savedData.LinkEntries[0].Status)
	}

	// Проверяем вторую задачу
	if savedData.LinkEntries[1].ID != 2 {
		t.Errorf("Expected ID 2, got %d", savedData.LinkEntries[1].ID)
	}
	if !reflect.DeepEqual(savedData.LinkEntries[1].URLs, links2) {
		t.Errorf("Expected URLs %v, got %v", links2, savedData.LinkEntries[1].URLs)
	}
}

// TestUpload проверяет загрузку задач из файла.
func TestUpload(t *testing.T) {
	// Создаем тестовые данные
	testData := storageData{
		NextID: 3,
		LinkEntries: []taskWithID{
			{
				ID:     1,
				URLs:   []string{"google.com", "yandex.ru"},
				Status: "available",
			},
			{
				ID:     2,
				URLs:   []string{"github.com"},
				Status: "",
			},
		},
	}

	// Создаем временный файл с тестовыми данными
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

	// Загружаем данные
	s := NewStorage()
	storage := getStorageLinkStatus(s)

	if err := s.Upload(tmpFile.Name()); err != nil {
		t.Fatalf("Upload() error = %v, want nil", err)
	}

	// Проверяем nextID
	if storage.nextID != 3 {
		t.Errorf("Expected nextID 3, got %d", storage.nextID)
	}

	// Проверяем количество задач
	if len(storage.Links) != 2 {
		t.Fatalf("Expected 2 tasks, got %d", len(storage.Links))
	}

	// Проверяем первую задачу
	if !reflect.DeepEqual(storage.Links[0].URLs, testData.LinkEntries[0].URLs) {
		t.Errorf("Expected URLs %v, got %v", testData.LinkEntries[0].URLs, storage.Links[0].URLs)
	}
	if storage.Links[0].Status != "available" {
		t.Errorf("Expected status 'available', got %s", storage.Links[0].Status)
	}

	// Проверяем вторую задачу
	if !reflect.DeepEqual(storage.Links[1].URLs, testData.LinkEntries[1].URLs) {
		t.Errorf("Expected URLs %v, got %v", testData.LinkEntries[1].URLs, storage.Links[1].URLs)
	}

	// Проверяем, что можно получить задачи по ID
	task1, err := s.Get(1)
	if err != nil {
		t.Errorf("Get(1) error = %v, want nil", err)
	}
	if !reflect.DeepEqual(task1.URLs, testData.LinkEntries[0].URLs) {
		t.Errorf("Get(1) URLs = %v, want %v", task1.URLs, testData.LinkEntries[0].URLs)
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
	// Создаем хранилище и добавляем задачи
	s1 := NewStorage()
	links1 := []string{"google.com", "yandex.ru"}
	links2 := []string{"github.com"}

	id1 := s1.Add(links1)
	id2 := s1.Add(links2)

	// Устанавливаем статусы
	storage1 := getStorageLinkStatus(s1)
	storage1.Links[id1-1].Status = "available"
	storage1.Links[id2-1].Status = "not available"

	// Сохраняем в файл
	tmpFile, err := os.CreateTemp("", "test_roundtrip_*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	if err := s1.Dump(tmpFile.Name()); err != nil {
		t.Fatalf("Dump() error = %v, want nil", err)
	}

	// Загружаем в новое хранилище
	s2 := NewStorage()
	if err := s2.Upload(tmpFile.Name()); err != nil {
		t.Fatalf("Upload() error = %v, want nil", err)
	}

	// Проверяем, что данные восстановились
	task1, err := s2.Get(1)
	if err != nil {
		t.Errorf("Get(1) error = %v, want nil", err)
	}
	if !reflect.DeepEqual(task1.URLs, links1) {
		t.Errorf("Get(1) URLs = %v, want %v", task1.URLs, links1)
	}
	if task1.Status != "available" {
		t.Errorf("Get(1) Status = %s, want 'available'", task1.Status)
	}

	task2, err := s2.Get(2)
	if err != nil {
		t.Errorf("Get(2) error = %v, want nil", err)
	}
	if !reflect.DeepEqual(task2.URLs, links2) {
		t.Errorf("Get(2) URLs = %v, want %v", task2.URLs, links2)
	}
	if task2.Status != "not available" {
		t.Errorf("Get(2) Status = %s, want 'not available'", task2.Status)
	}

	// Проверяем, что nextID восстановился
	storage2 := getStorageLinkStatus(s2)
	if storage2.nextID != 2 {
		t.Errorf("Expected nextID 2, got %d", storage2.nextID)
	}
}
