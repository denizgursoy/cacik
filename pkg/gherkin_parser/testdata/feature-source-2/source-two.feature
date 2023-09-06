Feature: Scenario Outline

  @Test
  Scenario: Data Table Scenario
    Given I verify the column contains expected value
      | columnName     | expectedValue     |
      | someColumnName | someExpectedValue |