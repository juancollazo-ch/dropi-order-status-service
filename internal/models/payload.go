package models

// WebhookPayload estructura simplificada para el webhook
type WebhookPayload struct {
    ID                  int64         `json:"id"`
    Status              string        `json:"status"`
    SupplierID          int64         `json:"supplier_id"`
    Dir                 string        `json:"dir"`
    Phone               string        `json:"phone"`
    Email               *string       `json:"email"`
    CreatedAt           string        `json:"created_at"`
    Type                string        `json:"type"`
    TotalOrder          string        `json:"total_order"`
    Notes               *string       `json:"notes"`
    Name                string        `json:"name"`
    Surname             string        `json:"surname"`
    Country             string        `json:"country"`
    State               string        `json:"state"`
    City                string        `json:"city"`
    ZipCode             *string       `json:"zip_code"`
    RateType            string        `json:"rate_type"`
    ShippingCompany     string        `json:"shipping_company"`
    ShippingGuide       string        `json:"shipping_guide"`
    Sticker             string        `json:"sticker"`
    SellerID            *int64        `json:"seller_id"`
    ShopOrderID         string        `json:"shop_order_id"`
    ShopID              int64         `json:"shop_id"`
    ShopOrderNumber     int           `json:"shop_order_number"`
    WarehouseID         int64         `json:"warehouse_id"`
    DNIType             *string       `json:"dni_type"`
    DNI                 *string       `json:"dni"`
    Colonia             *string       `json:"colonia"`
    ExternalID          *string       `json:"external_id"`
    Shop                WebhookShopInfo      `json:"shop"`
    NovedadServientrega *string              `json:"novedad_servientrega"`
    OrderDetails        []WebhookOrderDetail `json:"orderdetails"`
    Warehouse           WebhookWarehouseInfo `json:"warehouse"`
}

type WebhookShopInfo struct {
    ID                    int64   `json:"id"`
    UserID                int64   `json:"user_id"`
    Name                  string  `json:"name"`
    Email                 *string `json:"email"`
    Phone                 *string `json:"phone"`
    Type                  string  `json:"type"`
    CreatedAt             string  `json:"created_at"`
    UpdatedAt             string  `json:"updated_at"`
    DeletedAt             *string `json:"deleted_at"`
    ShopPassword          *string `json:"shop_password"`
    ChangeStatusPendiente bool    `json:"change_status_pendiente"`
    StatusPendiente       *string `json:"status_pendiente"`
    SyncShippingGuide     bool    `json:"sync_shipping_guide"`
    TypeID                int     `json:"type_id"`
    Webhook               *string `json:"webhook"`
}

type WebhookWarehouseInfo struct {
    ID   *int64  `json:"id"`
    Name *string `json:"name"`
}

type WebhookOrderDetail struct {
    ID      int64          `json:"id"`
    OrderID int64          `json:"order_id"`
    Price   string         `json:"price"`
    Product WebhookProduct `json:"product"`
}

type WebhookProduct struct {
    ID          int64  `json:"id"`
    IDLista     int    `json:"id_lista"`
    Name        string `json:"name"`
    NameInOrder string `json:"name_in_order"`
}

// ToWebhookPayload convierte un DropiOrder a WebhookPayload
// Esta función encapsula la lógica de conversión en el paquete models
func (order DropiOrder) ToWebhookPayload() WebhookPayload {
    return WebhookPayload{
        ID:                  order.ID,
        Status:              order.Status,
        SupplierID:          order.SupplierID,
        Dir:                 order.Dir,
        Phone:               order.Phone,
        Email:               order.Email,
        CreatedAt:           order.CreatedAt,
        Type:                order.Type,
        TotalOrder:          order.TotalOrder,
        Notes:               order.Notes,
        Name:                order.Name,
        Surname:             order.Surname,
        Country:             order.Country,
        State:               order.State,
        City:                order.City,
        ZipCode:             order.ZipCode,
        RateType:            order.RateType,
        ShippingCompany:     order.ShippingCompany,
        ShippingGuide:       order.ShippingGuide,
        Sticker:             getStickerValue(order.Sticker),
        SellerID:            order.SellerID,
        ShopOrderID:         order.ShopOrderID,
        ShopID:              order.ShopID,
        ShopOrderNumber:     int(order.ShopOrderNumber),
        WarehouseID:         order.WarehouseID,
        DNIType:             order.DNIType,
        DNI:                 order.DNI,
        Colonia:             order.Colonia,
        ExternalID:          order.ExternalID,
        Shop:                order.Shop.ToWebhookShopInfo(),
        NovedadServientrega: order.Novedad,
        OrderDetails:        order.ToWebhookOrderDetails(),
        Warehouse: WebhookWarehouseInfo{
            ID:   order.Warehouse.ID,
            Name: order.Warehouse.Name,
        },
    }
}

// ToWebhookOrderDetails convierte OrderDetails a WebhookOrderDetails
func (order DropiOrder) ToWebhookOrderDetails() []WebhookOrderDetail {
    if len(order.OrderDetails) == 0 {
        return []WebhookOrderDetail{}
    }

    webhookDetails := make([]WebhookOrderDetail, len(order.OrderDetails))
    for i, detail := range order.OrderDetails {
        webhookDetails[i] = WebhookOrderDetail{
            ID:      detail.ID,
            OrderID: detail.OrderID,
            Price:   detail.Price,
            Product: WebhookProduct{
                ID:          detail.Product.ID,
                IDLista:     detail.Product.IDLista,
                Name:        detail.Product.Name,
                NameInOrder: detail.Product.NameInOrder,
            },
        }
    }
    return webhookDetails
}

// getStickerValue convierte *string a string, retornando "" si es nil
func getStickerValue(sticker *string) string {
    if sticker == nil {
        return ""
    }
    return *sticker
}

// ToWebhookShopInfo convierte ShopInfo a WebhookShopInfo
func (shop ShopInfo) ToWebhookShopInfo() WebhookShopInfo {
    // Valores por defecto para campos opcionales
    var createdAt, updatedAt string
    var changePendiente, syncGuide bool
    var typeID int

    if shop.CreatedAt != nil {
        createdAt = *shop.CreatedAt
    }
    if shop.UpdatedAt != nil {
        updatedAt = *shop.UpdatedAt
    }
    if shop.ChangePendiente != nil {
        changePendiente = *shop.ChangePendiente
    }
    if shop.SyncGuide != nil {
        syncGuide = *shop.SyncGuide
    }
    if shop.TypeID != nil {
        typeID = *shop.TypeID
    }

    return WebhookShopInfo{
        ID:                    shop.ID,
        UserID:                shop.UserID,
        Name:                  shop.Name,
        Email:                 shop.Email,
        Phone:                 shop.Phone,
        Type:                  shop.Type,
        CreatedAt:             createdAt,
        UpdatedAt:             updatedAt,
        DeletedAt:             shop.DeletedAt,
        ShopPassword:          shop.ShopPassword,
        ChangeStatusPendiente: changePendiente,
        StatusPendiente:       shop.StatusPendiente,
        SyncShippingGuide:     syncGuide,
        TypeID:                typeID,
        Webhook:               shop.Webhook,
    }
}
