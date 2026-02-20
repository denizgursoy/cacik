Feature: Built-in parameter types

  Scenario: Using integer type
    Given I have 5 apples
    And I have -3 apples

  Scenario: Using float type
    Given the price is 19.99
    And the price is -0.5
    And the price is 100

  Scenario: Using word type
    Given my name is John
    And my name is test123

  Scenario: Using string type
    Given I say "Hello World"
    And I say "Testing with spaces and punctuation!"

  Scenario: Using any type
    Given I see anything at all here
    And I see 123 mixed content!
