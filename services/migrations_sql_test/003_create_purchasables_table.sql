-- Create the purchasables table with check constraints
CREATE TABLE purchasables (
  id UUID PRIMARY KEY,
  user_id UUID NOT NULL,
  name VARCHAR(255) NOT NULL,
  item_type VARCHAR(50) NOT NULL,
  cost NUMERIC NOT NULL,
  currency VARCHAR(3) NOT NULL, -- ISO 4217 currency code
  donation_ratio NUMERIC,
  inventory INTEGER,
  charge_recurrence_interval VARCHAR(20),
  charge_recurrence_interval_count INTEGER,
  charge_recurrence_end_date TIMESTAMP,
  created_at TIMESTAMP NOT NULL DEFAULT now(),
  updated_at TIMESTAMP NOT NULL DEFAULT now(),
  CONSTRAINT purchasables_item_type_check CHECK (item_type IN ('ticket', 'membership', 'donation', 'partialDonation', 'merchandise')),
  CONSTRAINT purchasables_charge_recurrence_interval_check CHECK (charge_recurrence_interval IN ('day', 'week', 'month', 'year')),
  CONSTRAINT purchasables_partial_donation_ratio_check CHECK ((item_type = 'partialDonation' AND donation_ratio IS NOT NULL) OR item_type != 'partialDonation')
);

-- Create an index on item_type for better query performance
CREATE INDEX purchasables_item_type_index ON purchasables (item_type);
CREATE INDEX purchasable_user_id_index ON purchasables (user_id);

