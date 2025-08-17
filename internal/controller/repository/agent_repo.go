package repository

import (
	"time"

	"github.com/xbox/sing-box-manager/internal/models"
	"gorm.io/gorm"
)

// AgentRepository Agent数据访问接口
type AgentRepository interface {
	// 创建Agent
	Create(agent *models.Agent) error
	// 根据ID获取Agent
	GetByID(id string) (*models.Agent, error)
	// 获取所有Agent列表
	GetAll(limit, offset int) ([]*models.Agent, int64, error)
	// 更新Agent
	Update(agent *models.Agent) error
	// 删除Agent
	Delete(id string) error
	// 更新Agent状态
	UpdateStatus(id string, status string) error
	// 更新心跳时间
	UpdateHeartbeat(id string) error
	// 获取在线Agent数量
	GetOnlineCount() (int64, error)
	// 获取离线Agent列表
	GetOfflineAgents(maxOfflineTime time.Duration) ([]*models.Agent, error)
	// 根据状态获取Agent列表
	GetByStatus(status string, limit, offset int) ([]*models.Agent, int64, error)
	// 获取所有Agent（无分页）
	GetAllAgents(ctx ...interface{}) ([]*models.Agent, error)
}

// agentRepository Agent数据访问实现
type agentRepository struct {
	db *gorm.DB
}

// NewAgentRepository 创建Agent数据访问实例
func NewAgentRepository(db *gorm.DB) AgentRepository {
	return &agentRepository{db: db}
}

// Create 创建Agent
func (r *agentRepository) Create(agent *models.Agent) error {
	return r.db.Create(agent).Error
}

// GetByID 根据ID获取Agent
func (r *agentRepository) GetByID(id string) (*models.Agent, error) {
	var agent models.Agent
	err := r.db.Where("id = ?", id).First(&agent).Error
	if err != nil {
		return nil, err
	}
	return &agent, nil
}

// GetAll 获取所有Agent列表
func (r *agentRepository) GetAll(limit, offset int) ([]*models.Agent, int64, error) {
	var agents []*models.Agent
	var total int64
	
	// 获取总数
	if err := r.db.Model(&models.Agent{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	
	// 获取分页数据
	query := r.db.Limit(limit).Offset(offset).Order("created_at DESC")
	if err := query.Find(&agents).Error; err != nil {
		return nil, 0, err
	}
	
	return agents, total, nil
}

// Update 更新Agent
func (r *agentRepository) Update(agent *models.Agent) error {
	return r.db.Save(agent).Error
}

// Delete 删除Agent
func (r *agentRepository) Delete(id string) error {
	return r.db.Delete(&models.Agent{}, "id = ?", id).Error
}

// UpdateStatus 更新Agent状态
func (r *agentRepository) UpdateStatus(id string, status string) error {
	return r.db.Model(&models.Agent{}).
		Where("id = ?", id).
		Update("status", status).Error
}

// UpdateHeartbeat 更新心跳时间
func (r *agentRepository) UpdateHeartbeat(id string) error {
	now := time.Now()
	return r.db.Model(&models.Agent{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"last_heartbeat": &now,
			"status":         "online",
		}).Error
}

// GetOnlineCount 获取在线Agent数量
func (r *agentRepository) GetOnlineCount() (int64, error) {
	var count int64
	err := r.db.Model(&models.Agent{}).
		Where("status = ?", "online").
		Count(&count).Error
	return count, err
}

// GetOfflineAgents 获取离线Agent列表
func (r *agentRepository) GetOfflineAgents(maxOfflineTime time.Duration) ([]*models.Agent, error) {
	var agents []*models.Agent
	cutoffTime := time.Now().Add(-maxOfflineTime)
	
	err := r.db.Where("last_heartbeat < ? OR last_heartbeat IS NULL", cutoffTime).
		Find(&agents).Error
	
	return agents, err
}

// GetByStatus 根据状态获取Agent列表
func (r *agentRepository) GetByStatus(status string, limit, offset int) ([]*models.Agent, int64, error) {
	var agents []*models.Agent
	var total int64
	
	// 获取总数
	if err := r.db.Model(&models.Agent{}).Where("status = ?", status).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	
	// 获取分页数据
	query := r.db.Where("status = ?", status).
		Limit(limit).Offset(offset).
		Order("created_at DESC")
	
	if err := query.Find(&agents).Error; err != nil {
		return nil, 0, err
	}
	
	return agents, total, nil
}

// GetAllAgents 获取所有Agent（无分页）
func (r *agentRepository) GetAllAgents(ctx ...interface{}) ([]*models.Agent, error) {
	var agents []*models.Agent
	err := r.db.Order("created_at DESC").Find(&agents).Error
	return agents, err
}