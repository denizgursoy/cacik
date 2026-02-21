Feature: Mixed parameter types

  Scenario: Want a colored vehicle with custom type, regex, int, and float
    When I want a red car with 4 doors costing 25000.50 dollars
    And I want a BLUE bike with 0 doors costing 999.99 dollars
    And I want a Green car with 2 doors costing 15000 dollars

  Scenario: Named item with color, string, and priority
    Given a red item named "Widget" at high priority
    And a BLUE item named "Gadget Pro" at 1 priority
    And a green item named "Super Item" at medium priority

  Scenario: Ownership with color, word, and boolean
    Then red owned by Alice is yes
    And blue owned by Bob is false
    And GREEN owned by Charlie is true

  Scenario: Sized item count with int, size, and color
    Given I have 5 small red boxes
    And I have 10 LARGE blue boxes
    And I have 3 Medium GREEN boxes

  Scenario: Product with all types combined
    When product SKU123 is red and small priced at 19.99 with high priority described as "A great product"
    And product ITEM456 is BLUE and LARGE priced at 99.50 with low priority described as "Budget option"

  Scenario: Quantity with any type
    Given I ordered 3 of red apples and some oranges
    And I ordered 100 of random stuff here

  Scenario: Conditional action with regex, color, and boolean
    When enable the red button and set active to true
    And disable the BLUE switch and set active to false
    And enable the green button and set active to yes

  Scenario: Case insensitivity for custom types
    # All these should work with case-insensitive matching
    When I want a RED car with 4 doors costing 20000 dollars
    And I want a red car with 4 doors costing 20000 dollars
    And I want a Red car with 4 doors costing 20000 dollars
    And I want a rEd car with 4 doors costing 20000 dollars

  Scenario: Priority by name and value
    Given a red item named "Test1" at high priority
    And a red item named "Test2" at 3 priority
    And a red item named "Test3" at LOW priority
    And a red item named "Test4" at 1 priority

  Scenario: All sizes case insensitive
    Given I have 1 SMALL red boxes
    And I have 1 small red boxes
    And I have 1 Small red boxes
    And I have 1 MEDIUM blue boxes
    And I have 1 medium blue boxes
    And I have 1 LARGE green boxes
    And I have 1 large green boxes
