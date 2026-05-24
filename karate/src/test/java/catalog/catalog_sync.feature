Feature: Catalog Sync API

  Background:
    * url baseUrl

  # ── GET /catalog/product-projections ────────────────────────────────────────

  Scenario: product-projections returns 200 with correct response shape
    Given path '/catalog/product-projections'
    When method GET
    Then status 200
    And match response.items == '#array'
    And match response.pageSize == '#number'
    And match response.hasNext == '#boolean'
    And match response.nextCursor == '#string'
    And match response.syncToken == '#string'

  Scenario: product-projections response fields use camelCase
    Given path '/catalog/product-projections'
    When method GET
    Then status 200
    And match response contains { items: '#array', pageSize: '#number', hasNext: '#boolean', nextCursor: '#string', syncToken: '#string' }

  Scenario: product-projections invalid updatedSince returns 400
    Given path '/catalog/product-projections'
    And param updatedSince = 'not-a-date'
    When method GET
    Then status 400
    And match response.error.code == 'INVALID_REQUEST'

  Scenario: product-projections invalid cursor returns 400
    Given path '/catalog/product-projections'
    And param cursor = '!!!invalid!!!'
    When method GET
    Then status 400
    And match response.error.code == 'INVALID_CURSOR'

  # ── GET /catalog/variant-inventory ──────────────────────────────────────────

  Scenario: variant-inventory returns 200 with correct response shape
    Given path '/catalog/variant-inventory'
    When method GET
    Then status 200
    And match response.items == '#array'
    And match response.pageSize == '#number'
    And match response.hasNext == '#boolean'
    And match response.nextCursor == '#string'
    And match response.syncToken == '#string'

  Scenario: variant-inventory pageSize=1 limits results
    Given path '/catalog/variant-inventory'
    And param pageSize = 1
    When method GET
    Then status 200
    And match response.pageSize == 1
    And assert response.items.length <= 1

  Scenario: variant-inventory cursor pagination
    # First page with pageSize=1
    Given path '/catalog/variant-inventory'
    And param pageSize = 1
    When method GET
    Then status 200
    * def page1 = response
    * match page1.items == '#array'
    * match page1.nextCursor == '#string'
    # Follow cursor (works whether or not hasNext is true — empty cursor treated as no cursor)
    Given path '/catalog/variant-inventory'
    And param pageSize = 1
    And param cursor = page1.nextCursor
    When method GET
    Then status 200
    And match response.items == '#array'
    And match response.hasNext == '#boolean'
    And match response.syncToken == '#string'

  Scenario: variant-inventory invalid updatedSince returns 400
    Given path '/catalog/variant-inventory'
    And param updatedSince = 'yesterday'
    When method GET
    Then status 400
    And match response.error.code == 'INVALID_REQUEST'

  Scenario: variant-inventory invalid cursor returns 400
    Given path '/catalog/variant-inventory'
    And param cursor = 'not-base64-json'
    When method GET
    Then status 400
    And match response.error.code == 'INVALID_CURSOR'

  Scenario: variant-inventory response fields use camelCase
    Given path '/catalog/variant-inventory'
    When method GET
    Then status 200
    And match response contains { items: '#array', pageSize: '#number', hasNext: '#boolean', nextCursor: '#string', syncToken: '#string' }

  Scenario: variant-inventory with valid updatedSince returns 200
    Given path '/catalog/variant-inventory'
    And param updatedSince = '2020-01-01T00:00:00Z'
    When method GET
    Then status 200
    And match response.items == '#array'
