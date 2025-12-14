package storage

import (
	"errors"
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
