//go:build integration

package repository_test

import (
	"context"
	"errors"
	"log"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/adapters/repository"
	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/adapters/repository/tables"
	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/app/query"
	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/domain"
)

var testDB *gorm.DB

func TestMain(m *testing.M) {
	os.Exit(run(m))
}

func run(m *testing.M) int {
	ctx := context.Background()
	container, err := tcpostgres.Run(ctx, "postgres:16", tcpostgres.BasicWaitStrategies())
	if err != nil {
		log.Printf("start postgres: %v", err)
		return 1
	}
	defer testcontainers.TerminateContainer(container)

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		log.Printf("connection string: %v", err)
		return 1
	}
	testDB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{TranslateError: true})
	if err != nil {
		log.Printf("open gorm: %v", err)
		return 1
	}
	if err := tables.Migrate(testDB); err != nil {
		log.Printf("migrate: %v", err)
		return 1
	}
	return m.Run()
}

func cleanDB(t *testing.T) {
	t.Helper()
	if err := testDB.Exec("TRUNCATE cards").Error; err != nil {
		t.Fatal(err)
	}
}

var baseTime = time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)

// Ids come from sequential uuid.NewV7() calls, which google/uuid keeps increasing, so id order is creation order.
func newCard(t *testing.T, minute int, last4, brand, holder string, expMonth, expYear int) *domain.Card {
	t.Helper()
	createdAt := baseTime.Add(time.Duration(minute) * time.Minute)
	card, err := domain.UnmarshalCardFromDatabase(
		domain.CardID(uuid.Must(uuid.NewV7()).String()),
		last4, expMonth, expYear, holder,
		domain.CardBrand(brand),
		uuid.NewString(),
		createdAt, createdAt,
	)
	if err != nil {
		t.Fatal(err)
	}
	return card
}

func seedCards(t *testing.T) []*domain.Card {
	t.Helper()
	cards := []*domain.Card{
		newCard(t, 1, "1111", "visa", "Ada Lovelace", 12, 2030),
		newCard(t, 2, "1111", "visa", "Adam Smith", 6, 2031),
		newCard(t, 3, "2222", "visa", "Grace Hopper", 3, 2033),
		newCard(t, 4, "1111", "mastercard", "Ada Byron", 12, 2030),
		newCard(t, 5, "1111", "visa", "A_b Underscore", 1, 2032),
		newCard(t, 6, "1111", "visa", "A%c Percent", 1, 2032),
	}
	repo := repository.NewCardRepository(testDB)
	for _, c := range cards {
		if err := repo.AddCard(context.Background(), c); err != nil {
			t.Fatal(err)
		}
	}
	return cards
}

func holders(cards []query.Card) []string {
	names := make([]string, len(cards))
	for i, c := range cards {
		names[i] = c.CardHolderName
	}
	return names
}

func TestAddAndGetCard(t *testing.T) {
	cleanDB(t)
	repo := repository.NewCardRepository(testDB)
	want := newCard(t, 1, "4242", "visa", "Ada Lovelace", 12, 2030)

	if err := repo.AddCard(context.Background(), want); err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetCard(context.Background(), want.ID())
	if err != nil {
		t.Fatal(err)
	}

	if got.ID != string(want.ID()) || got.Last4 != want.Last4() || got.Brand != string(want.Brand()) ||
		got.CardHolderName != want.CardHolderName() ||
		got.ExpMonth != want.ExpMonth() || got.ExpYear != want.ExpYear() ||
		!got.CreatedAt.Equal(want.CreatedAt()) || !got.UpdatedAt.Equal(want.UpdatedAt()) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestAddCardRejectsDuplicates(t *testing.T) {
	cleanDB(t)
	repo := repository.NewCardRepository(testDB)
	original := newCard(t, 1, "4242", "visa", "Ada Lovelace", 12, 2030)
	if err := repo.AddCard(context.Background(), original); err != nil {
		t.Fatal(err)
	}
	sameID, err := domain.UnmarshalCardFromDatabase(original.ID(), "1111", 1, 2031, "Other", "visa", "another-hash", baseTime, baseTime)
	if err != nil {
		t.Fatal(err)
	}
	samePANHash, err := domain.UnmarshalCardFromDatabase(
		domain.CardID(uuid.Must(uuid.NewV7()).String()), "1111", 1, 2031, "Other", "visa", original.PANHash(), baseTime, baseTime)
	if err != nil {
		t.Fatal(err)
	}

	for name, card := range map[string]*domain.Card{"same id": sameID, "same pan hash": samePANHash} {
		t.Run(name, func(t *testing.T) {
			err := repo.AddCard(context.Background(), card)

			if !errors.Is(err, domain.ErrCardAlreadyExists) {
				t.Fatalf("got %v, want ErrCardAlreadyExists", err)
			}
		})
	}
}

func TestGetCardNotFound(t *testing.T) {
	cleanDB(t)
	repo := repository.NewCardRepository(testDB)
	ids := map[string]domain.CardID{
		"unknown uuid": domain.CardID(uuid.Must(uuid.NewV7()).String()),
		"not a uuid":   "not-a-uuid",
	}

	for name, id := range ids {
		t.Run(name, func(t *testing.T) {
			if _, err := repo.GetCard(context.Background(), id); !errors.Is(err, domain.ErrCardNotFound) {
				t.Errorf("got %v, want ErrCardNotFound", err)
			}
		})
	}
}

func TestListCardsPaginates(t *testing.T) {
	cleanDB(t)
	cards := seedCards(t)
	reader := repository.NewCardRepository(testDB)

	var gotIDs []string
	cursor := ""
	pages := 0
	for {
		page, err := reader.ListCards(context.Background(), 2, cursor)
		if err != nil {
			t.Fatal(err)
		}
		pages++
		for _, c := range page.Items {
			gotIDs = append(gotIDs, c.ID)
		}
		if page.NextCursor == "" {
			break
		}
		cursor = page.NextCursor
	}

	var wantIDs []string
	for _, c := range cards {
		wantIDs = append(wantIDs, string(c.ID()))
	}
	if pages != 3 || !slices.Equal(gotIDs, wantIDs) {
		t.Fatalf("pages = %d, ids = %v, want 3 pages in order %v", pages, gotIDs, wantIDs)
	}
}

func TestInvalidCursor(t *testing.T) {
	cleanDB(t)
	reader := repository.NewCardRepository(testDB)

	_, listErr := reader.ListCards(context.Background(), 10, "!!!")
	_, searchErr := reader.SearchCards(context.Background(), query.CardFilter{Last4: "1111"}, 10, "!!!")

	if !errors.Is(listErr, domain.ErrInvalidCursor) || !errors.Is(searchErr, domain.ErrInvalidCursor) {
		t.Fatalf("list = %v, search = %v, want ErrInvalidCursor", listErr, searchErr)
	}
}

func TestSearchCards(t *testing.T) {
	cleanDB(t)
	seedCards(t)
	reader := repository.NewCardRepository(testDB)

	tests := []struct {
		name   string
		filter query.CardFilter
		want   []string
	}{
		{
			name:   "last4 only",
			filter: query.CardFilter{Last4: "2222"},
			want:   []string{"Grace Hopper"},
		},
		{
			name:   "holder prefix ignores case",
			filter: query.CardFilter{Last4: "1111", CardHolderName: "ada"},
			want:   []string{"Ada Lovelace", "Adam Smith", "Ada Byron"},
		},
		{
			name:   "brand and holder prefix",
			filter: query.CardFilter{Last4: "1111", Brand: "visa", CardHolderName: "ada"},
			want:   []string{"Ada Lovelace", "Adam Smith"},
		},
		{
			name:   "brand holder prefix and expiry",
			filter: query.CardFilter{Last4: "1111", Brand: "visa", CardHolderName: "ada", ExpMonth: 12, ExpYear: 2030},
			want:   []string{"Ada Lovelace"},
		},
		{
			name:   "holder is a prefix not a substring",
			filter: query.CardFilter{Last4: "1111", CardHolderName: "lovelace"},
			want:   []string{},
		},
		{
			name:   "underscore is not a wildcard",
			filter: query.CardFilter{Last4: "1111", CardHolderName: "A_"},
			want:   []string{"A_b Underscore"},
		},
		{
			name:   "percent is not a wildcard",
			filter: query.CardFilter{Last4: "1111", CardHolderName: "a%"},
			want:   []string{"A%c Percent"},
		},
		{
			name:   "exp month only",
			filter: query.CardFilter{Last4: "1111", ExpMonth: 1},
			want:   []string{"A_b Underscore", "A%c Percent"},
		},
		{
			name:   "exp year only",
			filter: query.CardFilter{Last4: "1111", ExpYear: 2031},
			want:   []string{"Adam Smith"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page, err := reader.SearchCards(context.Background(), tt.filter, 10, "")

			if err != nil {
				t.Fatal(err)
			}
			if got := holders(page.Items); !slices.Equal(got, tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSearchCardsPaginates(t *testing.T) {
	cleanDB(t)
	seedCards(t)
	reader := repository.NewCardRepository(testDB)
	filter := query.CardFilter{Last4: "1111"}

	var pageSizes []int
	cursor := ""
	for {
		page, err := reader.SearchCards(context.Background(), filter, 2, cursor)
		if err != nil {
			t.Fatal(err)
		}
		pageSizes = append(pageSizes, len(page.Items))
		if page.NextCursor == "" {
			break
		}
		cursor = page.NextCursor
	}

	if !slices.Equal(pageSizes, []int{2, 2, 1}) {
		t.Fatalf("page sizes = %v, want [2 2 1]", pageSizes)
	}
}
