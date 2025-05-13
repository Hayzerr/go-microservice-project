-- Удаляем расширение pgcrypto, оно нам больше не нужно
-- CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    price NUMERIC(10,2) NOT NULL,
    type VARCHAR(32) NOT NULL,
    stock INT NOT NULL,
    festival_id INT,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Добавим несколько базовых товаров
INSERT INTO products (name, description, price, type, stock)
VALUES
  ('VIP Ticket', 'Access to VIP zone', 150.00, 'TICKET', 1),
  ('Festival T-Shirt', 'Official festival merchandise', 25.00, 'MERCHANDISE', 100),
  ('Standard Ticket', 'General admission', 50.00, 'TICKET', 500);
