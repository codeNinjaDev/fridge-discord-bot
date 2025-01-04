package main

import (
	"encoding/json"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// FoodInfo struct with JSON serialization
type FoodInfo struct {
	gorm.Model
	FoodItem        string            `json:"food_item"`
	BarcodeNumber   string            `json:"barcode_number"`
	NutritionFacts  map[string]string `json:"nutrition_facts" gorm:"-"`
	SerializedFacts string            `json:"-" gorm:"column:nutrition_facts"` // Stored as string in DB
	Storage         string            `json:"storage"`
	RoomTemp        StorageInfo       `json:"room_temp" gorm:"embedded;embeddedPrefix:room_temp_"`
	Fridge          StorageInfo       `json:"fridge" gorm:"embedded;embeddedPrefix:fridge_"`
	FoodEmoji       string            `json:"food_emoji"`
	UserId          string            `json:"-" gorm:"index"`
	ImageUrl        string            `json:"-"`
}

// StorageInfo struct for room and fridge info
type StorageInfo struct {
	FoodSafetyWindow string `json:"food_safety_window"`
	Expiration       string `json:"expected_expiration_date"`
}

func (f *FoodInfo) BeforeSave(tx *gorm.DB) (err error) {
	if f.NutritionFacts != nil {
		serialized, err := json.Marshal(f.NutritionFacts)
		if err != nil {
			return err
		}
		f.SerializedFacts = string(serialized)
	}
	return nil
}

// FoodService struct to handle database operations
type FoodService struct {
	db *gorm.DB
}

// NewFoodService creates a new instance of FoodService
func NewFoodService(dsn string) (*FoodService, error) {
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// AutoMigrate schema
	if err := db.AutoMigrate(&FoodInfo{}); err != nil {
		return nil, err
	}

	return &FoodService{db: db}, nil
}

// CreateFood creates a new food entry
func (s *FoodService) CreateFood(food *FoodInfo) error {
	return s.db.Create(food).Error
}

func (s *FoodService) GetFood(userID string, id uint) (*FoodInfo, error) {
	var food FoodInfo
	// Filter by both userID (string) and food ID (uint)
	if err := s.db.Where("user_id = ? AND id = ?", userID, id).First(&food).Error; err != nil {
		return nil, err
	}
	return &food, nil
}

// GetAllFoods retrieves all food entries for a specific user
func (s *FoodService) GetAllFoods(userID string) ([]FoodInfo, error) {
	var foods []FoodInfo
	// Filter by userID (string)
	if err := s.db.Where("user_id = ?", userID).Find(&foods).Error; err != nil {
		return nil, err
	}
	return foods, nil
}

// UpdateFood updates an existing food entry
func (s *FoodService) UpdateFood(food *FoodInfo) error {
	return s.db.Save(food).Error
}

// DeleteFood deletes a food entry by ID
func (s *FoodService) DeleteFood(id uint) error {
	return s.db.Delete(&FoodInfo{}, id).Error
}

func (f *FoodInfo) AfterFind(tx *gorm.DB) (err error) {
	if f.SerializedFacts != "" {
		var facts map[string]string
		if err := json.Unmarshal([]byte(f.SerializedFacts), &facts); err != nil {
			return err
		}
		f.NutritionFacts = facts
	}
	return nil
}
