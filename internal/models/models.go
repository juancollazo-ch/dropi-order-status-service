package models

type Order struct {
	ID              int         `json:"id"`
	Status          string      `json:"status"`
	SupplierID      int         `json:"supplier_id"`
	Dir             string      `json:"dir"`
	Phone           string      `json:"phone"`
	Email           string      `json:"email"`
	CreatedAt       string      `json:"created_at"`
	Type            string      `json:"type"`
	TotalOrder      string      `json:"total_order"`
	Notes           string      `json:"notes"`
	Name            string      `json:"name"`
	Surname         string      `json:"surname"`
	Country         string      `json:"country"`
	State           string      `json:"state"`
	City            string      `json:"city"`
	ZipCode         string      `json:"zip_code"`
	RateType        string      `json:"rate_type"`
	ShippingCompany string      `json:"shipping_company"`
	ShippingGuide   string      `json:"shipping_guide"`
	Sticker         string      `json:"sticker"`
	SellerID        interface{} `json:"seller_id"`
	ShopOrderID     string      `json:"shop_order_id"`
	ShopID          int         `json:"shop_id"`
	ShopOrderNumber int         `json:"shop_order_number"`
	WarehouseID     int         `json:"warehouse_id"`
	DNIType         string      `json:"dni_type"`
	DNI             string      `json:"dni"`
	Colonia         string      `json:"colonia"`
	ExternalID      string      `json:"external_id"`
	Shop            Shop        `json:"shop"`
	Novedad         interface{} `json:"novedad_servientrega"`
	OrderDetails    []any       `json:"orderdetails"`
	Warehouse       Warehouse   `json:"warehouse"`
}

type Shop struct {
	ID              int         `json:"id"`
	UserID          int         `json:"user_id"`
	Name            string      `json:"name"`
	Email           string      `json:"email"`
	Phone           string      `json:"phone"`
	Type            string      `json:"type"`
	CreatedAt       string      `json:"created_at"`
	UpdatedAt       string      `json:"updated_at"`
	DeletedAt       interface{} `json:"deleted_at"`
	ShopPassword    interface{} `json:"shop_password"`
	ChangePendiente bool        `json:"change_status_pendiente"`
	StatusPendiente interface{} `json:"status_pendiente"`
	SyncGuide       bool        `json:"sync_shipping_guide"`
	TypeID          int         `json:"type_id"`
	Webhook         interface{} `json:"webhook"`
}

type Warehouse struct {
	ID   interface{} `json:"id"`
	Name interface{} `json:"name"`
}
