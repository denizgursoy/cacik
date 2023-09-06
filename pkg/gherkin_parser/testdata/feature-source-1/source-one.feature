Feature: Scenario

  Background: Before Test Scenarios
    Given I execute before step

  @Test
  Scenario: Scenario
    Given I use the parameterized step of "Scenario 1"


  @Test
  Scenario: Scenario with variables
    Given I use string "string", int 1, float 1.1 and boolean "false"

  @Rule1
  Rule: This is Rule 1
  - Description Line 1
  - Description Line 2

  Background: Rule 1 Background
    When I click on login link

  Scenario: Scenario 1
    Then I should see "Registration" link

  Scenario: Scenario 2
    Then I should see "Forgot Password" link