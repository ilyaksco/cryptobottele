package storage

import (
	"encoding/json"
	"fmt"

	"github.com/supabase-community/supabase-go"
	"github.com/supabase-community/postgrest-go"
)

type User struct {
	ID           int64  `json:"id"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name,omitempty"`
	Username     string `json:"username,omitempty"`
	LanguageCode string `json:"language_code"`
	Score        int64  `json:"score"`
}

type Storage struct {
	client *supabase.Client
}

func New(supabaseURL, supabaseKey string) (*Storage, error) {
	client, err := supabase.NewClient(supabaseURL, supabaseKey, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create supabase client: %w", err)
	}
	return &Storage{client: client}, nil
}

func (s *Storage) UpsertUser(user User) error {
	_, _, err := s.client.From("users").Upsert(user, "id", "*", "").Execute()
	return err
}

func (s *Storage) GetUser(userID int64) (*User, error) {
	var results []User
	data, _, err := s.client.From("users").Select("*", "exact", false).Eq("id", fmt.Sprintf("%d", userID)).Execute()
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &results); err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("user not found")
	}
	return &results[0], nil
}

func (s *Storage) UpdateUserLanguage(userID int64, langCode string) error {
	user, err := s.GetUser(userID)
	if err != nil {
		return fmt.Errorf("could not get user to update language: %w", err)
	}
	user.LanguageCode = langCode
	return s.UpsertUser(*user)
}

func (s *Storage) IncreaseUserScore(userID int64, points int) (int64, error) {
	user, err := s.GetUser(userID)
	if err != nil {
		return 0, fmt.Errorf("could not get user to update score: %w", err)
	}
	newScore := user.Score + int64(points)
	updateData := map[string]int64{"score": newScore}
	_, _, err = s.client.From("users").Update(updateData, "", "minimal").Eq("id", fmt.Sprintf("%d", userID)).Execute()
	if err != nil {
		return 0, err
	}
	return newScore, nil
}

func (s *Storage) GetTopUsers(limit int) ([]User, error) {
	var results []User
	orderOpts := postgrest.OrderOpts{
		Ascending: false,
	}
	data, _, err := s.client.From("users").Select("*", "exact", false).Order("score", &orderOpts).Limit(limit, "").Execute()
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &results); err != nil {
		return nil, err
	}
	return results, nil
}