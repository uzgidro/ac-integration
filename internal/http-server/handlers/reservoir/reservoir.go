package reservoir

import (
	"context"
	"database/sql"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	resp "integration/internal/lib/api/response"
	"integration/internal/lib/logger/sl"
	"integration/internal/storage/repo"
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

type DataFetcher interface {
	GetDailyDataCount(ctx context.Context, reservoirID int, dateFrom, dateTo string) (int64, error)
	GetDailyData(ctx context.Context, reservoirID int, dateFrom, dateTo string, page, perPage int) ([]repo.DailyDataRow, error)
}

// Response is the top-level structure for the JSON response.
type Response struct {
	Success     bool              `json:"success"`
	Status      int               `json:"status"`
	Msg         string            `json:"msg"`
	Title       string            `json:"title"`
	Description map[string]string `json:"description"`
	TotalCounts int64             `json:"total_counts"`
	Data        []DataItem        `json:"data"`
}

// DataItem represents a single record in the 'data' array.
// Note the omitempty tag for the conditional field.
type DataItem struct {
	ID               int     `json:"id"`
	SendDateTime     string  `json:"send_datetime"`
	ObjectName       string  `json:"object_name"`
	ObjectTIN        string  `json:"object_tin"`
	ObjectChief      string  `json:"object_chief"`
	ChiefPINFL       string  `json:"chief_pinfl"`
	UpperBefLevel    float64 `json:"upper_bef_level"`
	DownBefLevel     float64 `json:"down_bef_level"`
	UpperBefVolume   float64 `json:"upper_bef_volume"`
	UpperBefPressure float64 `json:"upper_bef_pressure,omitempty"`
}

// StaticData holds the static information for each reservoir.
type StaticData struct {
	Title        string
	ObjectName   string
	ObjectTIN    string
	ObjectChief  string
	ChiefPINFL   string
	DownBefLevel float64
	Description  map[string]string
}

// getStaticData is a helper function to encapsulate the static data logic.
func getStaticData(reservoirID int) (StaticData, bool) {
	desc := map[string]string{
		"id":               "id",
		"send_datetime":    "Маълумот юборилган сана ва вақт",
		"object_name":      "Сув омбори объекти номи",
		"object_tin":       "Сув омбори объекти СТИРи",
		"object_chief":     "Сув омбори объекти раҳбари ФИШ",
		"chief_pinfl":      "Сув омбори объекти раҳбари ПИНФЛи",
		"upper_bef_level":  "Юқори бьеф сув сатҳи  (метр)",
		"down_bef_level":   "Пастки бьеф сув сатҳи  (метр)",
		"upper_bef_volume": "Юқори бьеф сув ҳажми (млн.м3)",
	}

	switch reservoirID {
	case 1:
		return StaticData{
			Title:        "Андижон сув омборида жойлашган назорат-ўлчаш қурилмаларининг автоматлаштирилган-ташхис назорат тизими орқали келувчи маълумотларни олиш",
			ObjectName:   "Андижон сув омбори",
			ObjectTIN:    "304952767",
			ObjectChief:  "Мирзаев Фуркат Солохидинович",
			ChiefPINFL:   "32203821450019",
			DownBefLevel: 822,
			Description:  desc,
		}, true
	case 2:
		desc["upper_bef_pressure"] = "Юқори бьеф сув босими (КПа)"
		return StaticData{
			Title:        "Оҳангарон сув омборида жойлашган назорат-ўлчаш қурилмаларининг автоматлаштирилган-ташхис назорат тизими орқали келувчи маълумотларни олиш",
			ObjectName:   "Оҳангарон сув омбори",
			ObjectTIN:    "304952767",
			ObjectChief:  "Турдиев Ботир Бакирович",
			ChiefPINFL:   "31108620620016",
			DownBefLevel: 1010,
			Description:  desc,
		}, true
	case 4:
		desc["upper_bef_pressure"] = "Юқори бьеф сув босими (КПа)"
		return StaticData{
			Title:        "Ҳисорак сув омборида жойлашган назорат-ўлчаш қурилмаларининг автоматлаштирилган-ташхис назорат тизими орқали келувчи маълумотларни олиш",
			ObjectName:   "Ҳисорак сув омбори",
			ObjectTIN:    "304952767",
			ObjectChief:  "Зиядуллаев Салохиддин Файзуллаевич",
			ChiefPINFL:   "33101682730055",
			DownBefLevel: 1060,
			Description:  desc,
		}, true
	default:
		return StaticData{}, false
	}
}

// New is a handler factory. It now requires a repository to fetch data.
func New(log *slog.Logger, repository DataFetcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.reservoir.New"

		log := log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		// 1. Get and validate reservoirId from URL
		reservoirIDStr := chi.URLParam(r, "reservoirId")
		reservoirID, err := strconv.Atoi(reservoirIDStr)
		if err != nil {
			log.Error("invalid reservoirId format", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid reservoirId format"))
			return
		}

		// 2. Check if the reservoirId is supported
		staticData, ok := getStaticData(reservoirID)
		if !ok {
			log.Warn("unsupported reservoirId", slog.Int("id", reservoirID))
			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, resp.NotFound("unsupported reservoirId"))
			return
		}

		// 3. Get and validate query parameters
		dateFromStr := r.URL.Query().Get("date_from")
		dateToStr := r.URL.Query().Get("date_to")
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))

		if dateFromStr == "" || dateToStr == "" || page == 0 || perPage == 0 {
			log.Error("missing or invalid query parameters")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Missing required query parameters: date_from, date_to, page, per_page"))
			return
		}

		// 4. Fetch data from the database via the repository
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		totalRecords, err := repository.GetDailyDataCount(ctx, reservoirID, dateFromStr, dateToStr)
		if err != nil {
			log.Error("failed to get data count", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("failed to get data count"))
			return
		}

		dbData, err := repository.GetDailyData(ctx, reservoirID, dateFromStr, dateToStr, page, perPage)
		if err != nil && err != sql.ErrNoRows {
			log.Error("failed to get daily data", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("failed to get daily data"))
			return
		}

		// 5. Transform database data into the final response format
		dataItems := make([]DataItem, 0, len(dbData))
		for i, item := range dbData {
			dataItem := DataItem{
				ID:             (page-1)*perPage + i + 1, // Global ID based on pagination
				SendDateTime:   item.Date,
				ObjectName:     staticData.ObjectName,
				ObjectTIN:      staticData.ObjectTIN,
				ObjectChief:    staticData.ObjectChief,
				ChiefPINFL:     staticData.ChiefPINFL,
				UpperBefLevel:  item.Level,
				DownBefLevel:   staticData.DownBefLevel,
				UpperBefVolume: item.Volume,
			}
			if reservoirID == 2 || reservoirID == 4 {
				dataItem.UpperBefPressure = item.Level * 0.101
			}
			dataItems = append(dataItems, dataItem)
		}

		// 6. Assemble and send the final response
		response := Response{
			Success:     true,
			Status:      200,
			Msg:         "Сўровга асосан маълумотлар тўлиқ шакллантирилди",
			Title:       staticData.Title,
			Description: staticData.Description,
			TotalCounts: totalRecords,
			Data:        dataItems,
		}

		render.JSON(w, r, response)
	}
}
