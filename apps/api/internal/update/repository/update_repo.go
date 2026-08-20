package repository

import (
	"time"

	adminModel "kun-galgame-api/internal/admin/model"

	"gorm.io/gorm"
)

type UpdateRepository struct {
	db *gorm.DB
}

func NewUpdateRepository(db *gorm.DB) *UpdateRepository {
	return &UpdateRepository{db: db}
}

func (r *UpdateRepository) CountHistory() int64 {
	var total int64
	r.db.Model(&adminModel.UpdateLog{}).Count(&total)
	return total
}

func (r *UpdateRepository) FindHistoryPaginated(page, limit int) []adminModel.UpdateLog {
	var logs []adminModel.UpdateLog
	r.db.Order("created DESC").
		Offset((page - 1) * limit).Limit(limit).
		Find(&logs)
	return logs
}

func (r *UpdateRepository) CreateHistory(log *adminModel.UpdateLog) error {
	return r.db.Create(log).Error
}

func (r *UpdateRepository) UpdateHistory(id int, fields map[string]any) error {
	return r.db.Model(&adminModel.UpdateLog{}).Where("id = ?", id).
		Updates(fields).Error
}

func (r *UpdateRepository) DeleteHistory(id int) error {
	return r.db.Delete(&adminModel.UpdateLog{}, id).Error
}

func (r *UpdateRepository) todos(status *int) *gorm.DB {
	q := r.db.Model(&adminModel.Todo{})
	if status != nil {
		q = q.Where("status = ?", *status)
	}
	return q
}

func (r *UpdateRepository) CountTodos(status *int) int64 {
	var total int64
	r.todos(status).Count(&total)
	return total
}

func (r *UpdateRepository) FindTodosPaginated(page, limit int, status *int) []adminModel.Todo {
	var todos []adminModel.Todo
	r.todos(status).Order("created DESC").
		Offset((page - 1) * limit).Limit(limit).
		Find(&todos)
	return todos
}

func (r *UpdateRepository) CreateTodo(todo *adminModel.Todo) error {
	if todo.Status == 2 {
		now := time.Now()
		todo.CompletedTime = &now
	}
	return r.db.Create(todo).Error
}

func (r *UpdateRepository) FindTodoByID(id int) (*adminModel.Todo, error) {
	var todo adminModel.Todo
	err := r.db.First(&todo, id).Error
	if err != nil {
		return nil, err
	}
	return &todo, nil
}

func (r *UpdateRepository) ClaimTodo(id, userID int) error {
	return r.db.Model(&adminModel.Todo{}).Where("id = ?", id).
		Updates(map[string]any{"status": 1, "claimed_user_id": userID}).Error
}

func (r *UpdateRepository) CompleteTodo(id int) error {
	return r.db.Model(&adminModel.Todo{}).Where("id = ?", id).
		Updates(map[string]any{"status": 2, "completed_time": time.Now()}).Error
}

func (r *UpdateRepository) DiscardTodo(id int) error {
	return r.db.Model(&adminModel.Todo{}).Where("id = ?", id).
		Updates(map[string]any{"status": 3, "completed_time": nil}).Error
}

func (r *UpdateRepository) UpdateTodo(id int, fields map[string]any) error {
	return r.db.Model(&adminModel.Todo{}).Where("id = ?", id).
		Updates(fields).Error
}

func (r *UpdateRepository) DeleteTodo(id int) error {
	return r.db.Delete(&adminModel.Todo{}, id).Error
}
