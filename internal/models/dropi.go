package models

type DropiOrder struct {
	ID              int64    `json:"id"`
	Status          string   `json:"status"`
	SupplierID      int64    `json:"supplier_id"`
	Dir             string   `json:"dir"`
	Phone           string   `json:"phone"`
	Email           *string  `json:"client_email"`
	CreatedAt       string   `json:"created_at"`
	Type            string   `json:"type"`
	TotalOrder      string   `json:"total_order"`
	Notes           *string  `json:"notes"`
	Name            string   `json:"name"`
	Surname         string   `json:"surname"`
	Country         string   `json:"country"`
	State           string   `json:"state"`
	City            string   `json:"city"`
	ZipCode         *string  `json:"zip_code"`
	RateType        string   `json:"rate_type"`
	ShippingCompany string   `json:"shipping_company"`
	ShippingGuide   string   `json:"shipping_guide"`
	Sticker         *string  `json:"sticker"`

	// 🔥 REFACTORIZADO (antes interface{})
	SellerID *int64 `json:"seller_id"`

	ShopOrderID     string  `json:"shop_order_id"`
	ShopID          int64   `json:"shop_id"`
	ShopOrderNumber int64   `json:"shop_order_number"`
	WarehouseID     int64   `json:"warehouse_id"`
	DNIType         *string `json:"dni_type"`
	DNI             *string `json:"dni"`
	Colonia         *string `json:"colonia"`
	ExternalID      *string `json:"external_id"`

	Shop ShopInfo `json:"shop"`

	// 🔥 REFACTORIZADO (antes interface{})
	Novedad *string `json:"novedad_servientrega"`

	// 🔥 REFACTORIZADO (antes []interface{})
	OrderDetails []OrderDetail `json:"orderdetails"`

	// warehouse también refactorizado
	Warehouse WarehouseInfo `json:"warehouse"`

	History []HistoryItem `json:"history"`
}

type ShopInfo struct {
	ID     int64  `json:"id"`
	UserID int64  `json:"user_id"`
	Name   string `json:"name"`
	Type   string `json:"type"`

	// Campos opcionales que pueden no venir en todas las respuestas
	Email           *string `json:"email,omitempty"`
	Phone           *string `json:"phone,omitempty"`
	CreatedAt       *string `json:"created_at,omitempty"`
	UpdatedAt       *string `json:"updated_at,omitempty"`
	DeletedAt       *string `json:"deleted_at,omitempty"`
	ShopPassword    *string `json:"shop_password,omitempty"`
	ChangePendiente *bool   `json:"change_status_pendiente,omitempty"`
	StatusPendiente *string `json:"status_pendiente,omitempty"`
	SyncGuide       *bool   `json:"sync_shipping_guide,omitempty"`
	TypeID          *int    `json:"type_id,omitempty"`
	Webhook         *string `json:"webhook,omitempty"`
}

type WarehouseInfo struct {
	ID   *int64  `json:"id"`
	Name *string `json:"name"`
}

type OrderDetail struct {
	ID      int64   `json:"id"`
	OrderID int64   `json:"order_id"`
	Price   string  `json:"price"`
	Product Product `json:"product"`
}

type Product struct {
	ID          int64  `json:"id"`
	IDLista     int    `json:"id_lista"`
	Name        string `json:"name"`
	NameInOrder string `json:"name_in_order"`
}

type HistoryItem struct {
	ID        int64         `json:"id"`
	OrderID   int64         `json:"order_id"`
	Status    string        `json:"status"`
	CreatedAt string        `json:"created_at"`
	User      *UserInfo     `json:"user,omitempty"`
	ChatUser  *ChatUserInfo `json:"usuario_chatcenter,omitempty"`
	Guide     *string       `json:"shipping_guide,omitempty"`
}

type UserInfo struct {
	ID       int64    `json:"id"`
	Name     string   `json:"name"`
	Surname  string   `json:"surname"`
	RoleUser RoleUser `json:"role_user"`
}

type RoleUser struct {
	Name string `json:"name"`
}

type ChatUserInfo struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// DropiAPIResponse representa la respuesta de la API de Dropi
type DropiAPIResponse struct {
	Objects []DropiOrder `json:"objects"`
	Count   int          `json:"count,omitempty"`
	Next    string       `json:"next,omitempty"`
}

// GetProductNames extrae los nombres de todos los productos de una orden
func (order *DropiOrder) GetProductNames() []string {
	if len(order.OrderDetails) == 0 {
		return []string{}
	}

	names := make([]string, 0, len(order.OrderDetails))
	for _, detail := range order.OrderDetails {
		if detail.Product.Name != "" {
			names = append(names, detail.Product.Name)
		}
	}
	return names
}
