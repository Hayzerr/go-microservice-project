-- ./order-service/db/init.sql
CREATE TABLE IF NOT EXISTS orders (
                                      id UUID PRIMARY KEY,
                                      user_id UUID NOT NULL,
                                      product_id UUID NOT NULL,
                                      quantity INTEGER NOT NULL,
                                      status TEXT NOT NULL,
                                      created_at TIMESTAMPTZ DEFAULT NOW()
    );
