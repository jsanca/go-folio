Feature: Product REST API Contract

  # Tests assume the Go API is running at baseUrl (default: http://localhost:8080).
  # All JSON fields use camelCase to match Jackson's default naming convention.
  # Each scenario generates a unique SKU/slug via UUID to be independently re-runnable.

  Background:
    * url baseUrl

  # ---------------------------------------------------------------------------
  # HAPPY PATH: full product lifecycle in a single scenario so state flows
  # naturally (create → read → update → deactivate → activate → delete).
  # ---------------------------------------------------------------------------

  Scenario: Full product lifecycle
    * def suffix = java.util.UUID.randomUUID() + ''
    * def sku    = 'BAG-' + suffix
    * def slug   = 'classic-leather-bag-' + suffix

    # 1. Create product — expect 201 with all camelCase fields populated
    Given path '/products'
    And header Content-Type = 'application/json'
    And request
      """
      {
        "sku": "#(sku)",
        "externalSystemId": "SAP-001",
        "title": "Classic Leather Bag",
        "slug": "#(slug)",
        "shortDescription": "A classic handcrafted leather bag.",
        "description": "A premium leather bag made for everyday use.",
        "category": "Bags",
        "tags": ["leather", "bag", "handmade"],
        "mainImageUrl": "https://example.com/images/classic-leather-bag.jpg",
        "retailPrice": { "amountCents": 19999 },
        "salePrice":   { "amountCents": 14999 },
        "currency": "USD",
        "stockQuantity": 10,
        "stockStatus": "IN_STOCK",
        "warehouseCode": "WH-001",
        "active": true
      }
      """
    When method POST
    Then status 201
    And match response.id             == '#number'
    And match response.sku            == sku
    And match response.title          == 'Classic Leather Bag'
    And match response.slug           == slug
    And match response.retailPrice.amountCents == 19999
    And match response.salePrice.amountCents   == 14999
    And match response.currency       == 'USD'
    And match response.stockQuantity  == 10
    And match response.stockStatus    == 'IN_STOCK'
    And match response.active         == true
    And match response.createdAt      == '#string'
    And match response.updatedAt      == '#string'
    * def productId = response.id

    # 2. Get product by ID — expect 200 with same data
    Given path '/products/' + productId
    When method GET
    Then status 200
    And match response.id  == productId
    And match response.sku == sku

    # 3. Get product by SKU — expect 200
    Given path '/products/sku/' + sku
    When method GET
    Then status 200
    And match response.sku == sku

    # 4. Update inventory — expect 200 with updated stockQuantity and stockStatus
    Given path '/products/sku/' + sku + '/inventory'
    And header Content-Type = 'application/json'
    And request { "quantity": 3, "stockStatus": "LOW_STOCK" }
    When method PATCH
    Then status 200
    And match response.stockQuantity == 3
    And match response.stockStatus   == 'LOW_STOCK'

    # 5. Update pricing — expect 200 with updated price fields
    Given path '/products/sku/' + sku + '/pricing'
    And header Content-Type = 'application/json'
    And request
      """
      {
        "retailPrice": { "amountCents": 24999 },
        "salePrice":   { "amountCents": 19999 },
        "currency": "EUR"
      }
      """
    When method PATCH
    Then status 200
    And match response.retailPrice.amountCents == 24999
    And match response.salePrice.amountCents   == 19999
    And match response.currency == 'EUR'

    # 6. Deactivate product — expect 204 with empty body
    Given path '/products/' + productId + '/deactivate'
    When method PATCH
    Then status 204
    And match response == ''

    # 7. Activate product — expect 204 with empty body
    Given path '/products/' + productId + '/activate'
    When method PATCH
    Then status 204
    And match response == ''

    # 8. Delete product — expect 204 with empty body
    Given path '/products/' + productId
    When method DELETE
    Then status 204
    And match response == ''

  # ---------------------------------------------------------------------------
  # LIST
  # ---------------------------------------------------------------------------

  Scenario: List products returns a JSON array
    Given path '/products'
    When method GET
    Then status 200
    And match response == '#array'

  # ---------------------------------------------------------------------------
  # ERROR CASES
  # ---------------------------------------------------------------------------

  Scenario: Create product with missing SKU returns 400 INVALID_PRODUCT
    * def suffix = java.util.UUID.randomUUID() + ''
    Given path '/products'
    And header Content-Type = 'application/json'
    And request
      """
      {
        "title": "Bag Without SKU",
        "slug": "bag-without-sku-#(suffix)",
        "currency": "USD",
        "retailPrice": { "amountCents": 1000 },
        "stockQuantity": 1,
        "stockStatus": "IN_STOCK"
      }
      """
    When method POST
    Then status 400
    And match response.error.code    == 'INVALID_PRODUCT'
    And match response.error.message == '#string'

  Scenario: Create product with duplicate SKU returns 409 DUPLICATE_SKU
    * def suffix = java.util.UUID.randomUUID() + ''
    * def sku   = 'DUP-SKU-' + suffix
    * def slug1 = 'dup-sku-first-'  + suffix
    * def slug2 = 'dup-sku-second-' + suffix

    # First product
    Given path '/products'
    And header Content-Type = 'application/json'
    And request
      """
      {
        "sku": "#(sku)",
        "title": "First Bag",
        "slug": "#(slug1)",
        "currency": "USD",
        "retailPrice": { "amountCents": 1000 },
        "stockQuantity": 1,
        "stockStatus": "IN_STOCK"
      }
      """
    When method POST
    Then status 201

    # Second product with same SKU — must conflict
    Given path '/products'
    And header Content-Type = 'application/json'
    And request
      """
      {
        "sku": "#(sku)",
        "title": "Second Bag",
        "slug": "#(slug2)",
        "currency": "USD",
        "retailPrice": { "amountCents": 2000 },
        "stockQuantity": 2,
        "stockStatus": "IN_STOCK"
      }
      """
    When method POST
    Then status 409
    And match response.error.code == 'DUPLICATE_SKU'

  Scenario: Create product with duplicate slug returns 409 DUPLICATE_SLUG
    * def suffix = java.util.UUID.randomUUID() + ''
    * def sku1 = 'DUP-SLUG-A-' + suffix
    * def sku2 = 'DUP-SLUG-B-' + suffix
    * def slug = 'dup-slug-' + suffix

    # First product
    Given path '/products'
    And header Content-Type = 'application/json'
    And request
      """
      {
        "sku": "#(sku1)",
        "title": "First Bag",
        "slug": "#(slug)",
        "currency": "USD",
        "retailPrice": { "amountCents": 1000 },
        "stockQuantity": 1,
        "stockStatus": "IN_STOCK"
      }
      """
    When method POST
    Then status 201

    # Second product with same slug — must conflict
    Given path '/products'
    And header Content-Type = 'application/json'
    And request
      """
      {
        "sku": "#(sku2)",
        "title": "Second Bag",
        "slug": "#(slug)",
        "currency": "USD",
        "retailPrice": { "amountCents": 2000 },
        "stockQuantity": 2,
        "stockStatus": "IN_STOCK"
      }
      """
    When method POST
    Then status 409
    And match response.error.code == 'DUPLICATE_SLUG'

  Scenario: Get non-existing product by ID returns 404 PRODUCT_NOT_FOUND
    Given path '/products/999999999'
    When method GET
    Then status 404
    And match response.error.code    == 'PRODUCT_NOT_FOUND'
    And match response.error.message == '#string'

  Scenario: Malformed JSON returns 400 INVALID_JSON
    Given path '/products'
    And header Content-Type = 'application/json'
    And request '{broken json'
    When method POST
    Then status 400
    And match response.error.code == 'INVALID_JSON'

  Scenario: Unknown JSON field returns 400 INVALID_JSON
    Given path '/products'
    And header Content-Type = 'application/json'
    And request
      """
      {
        "unknownField": "should-cause-error",
        "anotherUnknown": 42
      }
      """
    When method POST
    Then status 400
    And match response.error.code == 'INVALID_JSON'
