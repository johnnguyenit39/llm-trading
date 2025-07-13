package model

import (
	emailType "j_ai_trade/common"
)

type SendEmailRequest struct {
	Subject    string              `json:"subject" binding:"required"`
	Content    string              `json:"content" binding:"required"`
	Receivers  []string            `json:"receivers" binding:"required,dive,email"`
	Type       emailType.EmailType `json:"Type" binding:"required"`
	DataMapper []DataMapper        `json:"data_mapper"`
}

type DataMapperType string

const (
	DataMapperTypeText     DataMapperType = "text"
	DataMapperTypeImageUrl DataMapperType = "image_url"
)

type DataMapper struct {
	Field string         `json:"field"`
	Type  DataMapperType `json:"type"`
	Value string         `json:"value"`
}
