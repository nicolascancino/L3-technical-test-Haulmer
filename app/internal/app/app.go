package app

import (
	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/app/command"
	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/app/query"
)

type Application struct {
	Commands Commands
	Queries  Queries
}

type Commands struct {
	CreateCard command.CreateCardHandler
}

type Queries struct {
	GetCard     query.GetCardHandler
	ListCards   query.ListCardsHandler
	SearchCards query.SearchCardsHandler
}
