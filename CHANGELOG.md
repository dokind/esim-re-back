# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]
- TBD

## [2025-08-11] Package Pricing & API Field Renames
### Added
- PackagePrice model and per-package pricing synchronization with provider packages.
- Enriched detailed packages endpoint returning pricing metadata (effective USD/MNT, source, markup, override).
- Order flow now requires package_price_id or provider_price_id and links orders to selected PackagePrice.
- Effective MNT pricing calculations for PackagePrice (sync, markup, override).
- Swagger regeneration including enriched package DTOs and new order request fields.

### Changed (Breaking)
- Product field `custom_price` renamed to `custom_price_usd` (represents USD override before FX conversion).
- Order creation field `custom_price` renamed to `custom_price_usd` accordingly.

### Migration Notes
If you had existing data in the old `custom_price` column (auto-migrated will create new `custom_price_usd` column), run:
```
UPDATE products SET custom_price_usd = custom_price WHERE custom_price_usd IS NULL;
```
(Optional) Drop old column after verifying data copy (if still present depending on auto-migration behavior):
```
-- Verify column existence first
ALTER TABLE products DROP COLUMN IF EXISTS custom_price;
```

No data loss expected for new columns (PackagePrice will populate after running package sync endpoint).

### Next Steps
- Add tests for order creation with package pricing.
- Provide formal DTO for public product list including min package price.
- Consider background job to refresh exchange rates & recompute EffectivePriceMNT.
