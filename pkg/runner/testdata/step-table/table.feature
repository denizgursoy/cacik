Feature: DataTable support

  Scenario: Create users from a table
    Given the following users:
      | name  | age |
      | Alice | 30  |
      | Bob   | 25  |
    Then there should be 2 users

  Scenario: Add items with count
    Given I have 3 items:
      | item   | price |
      | apple  | 1.50  |
      | banana | 0.75  |
      | cherry | 2.00  |

  Scenario: Plot coordinates
    Given the coordinates are:
      | 10 | 20 |
      | 30 | 40 |
      | 50 | 60 |
