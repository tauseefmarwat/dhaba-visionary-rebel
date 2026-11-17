package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Server struct {
	db           *pgxpool.Pool
	rdb          *redis.Client
	version, pod string
}
type CreateOrder struct {
	CustomerName string           `json:"customer_name"`
	Items        []OrderItemInput `json:"items"`
}
type OrderItemInput struct {
	MenuItemID int64 `json:"menu_item_id"`
	Quantity   int   `json:"quantity"`
}
type OrderItem struct {
	MenuItemID int64   `json:"menu_item_id"`
	ItemName   string  `json:"item_name"`
	Quantity   int     `json:"quantity"`
	UnitPrice  float64 `json:"unit_price"`
	Subtotal   float64 `json:"subtotal"`
}
type Order struct {
	ID           int64       `json:"id"`
	OrderNumber  string      `json:"order_number"`
	CustomerName string      `json:"customer_name"`
	TotalAmount  float64     `json:"total_amount"`
	Currency     string      `json:"currency"`
	Status       string      `json:"status"`
	CreatedAt    time.Time   `json:"created_at"`
	Items        []OrderItem `json:"items"`
}

func main() {
	ctx := context.Background()
	dsn := env("DATABASE_URL", "postgres://dhaba:dhaba_dev_password@localhost:5432/dhaba_orders?sslmode=disable")
	db, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	r := redis.NewClient(&redis.Options{Addr: env("REDIS_ADDR", "localhost:6379"), Password: os.Getenv("REDIS_PASSWORD")})
	s := &Server{db: db, rdb: r, version: env("APP_VERSION", "dev"), pod: env("POD_NAME", "local")}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("GET /info", s.info)
	mux.HandleFunc("POST /api/orders", s.create)
	mux.HandleFunc("GET /api/orders", s.list)
	mux.HandleFunc("GET /api/orders/", s.get)
	port := env("PORT", "8002")
	log.Printf("orders-api listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, logging(mux)))
}
func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
func write(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	ctx, c := context.WithTimeout(r.Context(), 2*time.Second)
	defer c()
	if err := s.db.Ping(ctx); err != nil {
		write(w, 503, map[string]string{"status": "unhealthy"})
		return
	}
	if err := s.rdb.Ping(ctx).Err(); err != nil {
		write(w, 503, map[string]string{"status": "unhealthy"})
		return
	}
	write(w, 200, map[string]string{"status": "ok", "service": "orders-api"})
}
func (s *Server) info(w http.ResponseWriter, r *http.Request) {
	write(w, 200, map[string]string{"service": "orders-api", "version": s.version, "pod": s.pod})
}
func (s *Server) create(w http.ResponseWriter, r *http.Request) {
	var in CreateOrder
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || strings.TrimSpace(in.CustomerName) == "" || len(in.Items) == 0 {
		write(w, 400, map[string]string{"error": "customer_name and at least one item are required"})
		return
	}
	ctx := r.Context()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		write(w, 500, map[string]string{"error": "database error"})
		return
	}
	defer tx.Rollback(ctx)
	var orderID int64
	var orderNo string
	var total float64
	err = tx.QueryRow(ctx, "INSERT INTO orders(order_number,customer_name,total_amount,status) VALUES('PENDING',$1,0,'new') RETURNING id", strings.TrimSpace(in.CustomerName)).Scan(&orderID)
	if err != nil {
		write(w, 500, map[string]string{"error": "could not create order"})
		return
	}
	orderNo = fmt.Sprintf("DH-%06d", orderID)
	if _, err = tx.Exec(ctx, "UPDATE orders SET order_number=$1 WHERE id=$2", orderNo, orderID); err != nil {
		write(w, 500, map[string]string{"error": "could not number order"})
		return
	}
	for _, it := range in.Items {
		if it.MenuItemID <= 0 || it.Quantity < 1 || it.Quantity > 50 {
			write(w, 400, map[string]string{"error": "invalid item or quantity"})
			return
		}
		var name string
		var price float64
		err = tx.QueryRow(ctx, "SELECT name,price FROM menu_items WHERE id=$1 AND available=TRUE", it.MenuItemID).Scan(&name, &price)
		if err != nil {
			write(w, 400, map[string]string{"error": "one or more dishes are unavailable"})
			return
		}
		sub := price * float64(it.Quantity)
		total += sub
		_, err = tx.Exec(ctx, "INSERT INTO order_items(order_id,menu_item_id,item_name,quantity,unit_price,subtotal) VALUES($1,$2,$3,$4,$5,$6)", orderID, it.MenuItemID, name, it.Quantity, price, sub)
		if err != nil {
			write(w, 500, map[string]string{"error": "could not save order items"})
			return
		}
	}
	if _, err = tx.Exec(ctx, "UPDATE orders SET total_amount=$1 WHERE id=$2", total, orderID); err != nil {
		write(w, 500, map[string]string{"error": "could not total order"})
		return
	}
	if _, err = tx.Exec(ctx, "INSERT INTO order_status_history(order_id,new_status) VALUES($1,'new')", orderID); err != nil {
		write(w, 500, map[string]string{"error": "could not record order status"})
		return
	}
	if err = tx.Commit(ctx); err != nil {
		write(w, 500, map[string]string{"error": "could not commit order"})
		return
	}
	// Queue after DB commit. Production hardening should reconcile orders if Redis is unavailable.
	payload, _ := json.Marshal(map[string]int64{"order_id": orderID})
	if err = s.rdb.LPush(ctx, "dhaba:orders:queue", payload).Err(); err != nil {
		write(w, 503, map[string]string{"error": "order saved but queue unavailable; contact staff with order number " + orderNo})
		return
	}
	minute := time.Now().UTC().Format("200601021504")
	_ = s.rdb.Incr(ctx, "dhaba:stats:orders:"+minute).Err()
	_ = s.rdb.Expire(ctx, "dhaba:stats:orders:"+minute, 3*time.Hour).Err()
	rows, _ := s.db.Query(ctx, "SELECT item_name,quantity FROM order_items WHERE order_id=$1", orderID)
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var n string
			var q int
			if rows.Scan(&n, &q) == nil {
				_ = s.rdb.ZIncrBy(ctx, "dhaba:stats:dishes", float64(q), n).Err()
			}
		}
	}
	o, err := s.loadOrder(ctx, orderID)
	if err != nil {
		write(w, 201, map[string]any{"order_id": orderID, "order_number": orderNo, "status": "new"})
		return
	}
	write(w, 201, map[string]any{"order": o})
}
func (s *Server) loadOrder(ctx context.Context, id int64) (Order, error) {
	var o Order
	err := s.db.QueryRow(ctx, "SELECT id,order_number,customer_name,total_amount,currency,status,created_at FROM orders WHERE id=$1", id).Scan(&o.ID, &o.OrderNumber, &o.CustomerName, &o.TotalAmount, &o.Currency, &o.Status, &o.CreatedAt)
	if err != nil {
		return o, err
	}
	o.Items = []OrderItem{}
	rows, err := s.db.Query(ctx, "SELECT menu_item_id,item_name,quantity,unit_price,subtotal FROM order_items WHERE order_id=$1", id)
	if err != nil {
		return o, err
	}
	defer rows.Close()
	for rows.Next() {
		var i OrderItem
		if err = rows.Scan(&i.MenuItemID, &i.ItemName, &i.Quantity, &i.UnitPrice, &i.Subtotal); err != nil {
			return o, err
		}
		o.Items = append(o.Items, i)
	}
	return o, rows.Err()
}
func (s *Server) get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(strings.TrimPrefix(r.URL.Path, "/api/orders/"), 10, 64)
	if err != nil || id < 1 {
		write(w, 400, map[string]string{"error": "invalid order id"})
		return
	}
	o, err := s.loadOrder(r.Context(), id)
	if err != nil {
		write(w, 404, map[string]string{"error": "order not found"})
		return
	}
	write(w, 200, map[string]any{"order": o})
}
func (s *Server) list(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query(r.Context(), "SELECT id FROM orders ORDER BY created_at DESC LIMIT 100")
	if err != nil {
		write(w, 500, map[string]string{"error": "database error"})
		return
	}
	defer rows.Close()
	out := []Order{}
	for rows.Next() {
		var id int64
		if rows.Scan(&id) == nil {
			o, e := s.loadOrder(r.Context(), id)
			if e == nil {
				out = append(out, o)
			}
		}
	}
	write(w, 200, map[string]any{"orders": out})
}
func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}
