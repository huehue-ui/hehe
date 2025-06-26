-- Drop the index first if it exists
DROP INDEX IF EXISTS idx_targeting_lookup;

-- Drop the targeting_rules table first due to the foreign key constraint
DROP TABLE IF EXISTS targeting_rules;

-- Drop the campaigns table
DROP TABLE IF EXISTS campaigns;
