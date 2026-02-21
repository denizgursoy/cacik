Feature: Float parameter type

  Scenario: Positive float values
    Given the item costs 19.99 dollars
    And the item costs 100.00 dollars

  Scenario: Negative float values
    Given the temperature is -5.5 degrees
    And the temperature is -0.1 degrees

  Scenario: Integer as float
    Given the item costs 50 dollars
    And the temperature is 0 degrees
