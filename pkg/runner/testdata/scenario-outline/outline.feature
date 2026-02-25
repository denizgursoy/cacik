@outline
Feature: Scenario Outline processing

  Background:
    Given the application is started

  # Basic outline: multiple placeholders, multiple rows
  Scenario Outline: User login
    Given user "<username>" exists with role "<role>"
    When user "<username>" logs in with password "<password>"
    Then the login result should be "<result>"
    And the user role should be "<role>"

    Examples: Valid credentials
      | username | password | role    | result  |
      | alice    | secret1  | admin   | success |
      | bob      | secret2  | editor  | success |
      | charlie  | secret3  | viewer  | success |

    @negative
    Examples: Invalid credentials
      | username | password | role   | result  |
      | alice    | wrong    | admin  | failure |
      | unknown  | any      | none   | failure |

  # Outline with a DataTable in the step (placeholders inside the DataTable)
  Scenario Outline: Create user with permissions
    Given user "<username>" exists with role "<role>"
    When I assign permissions to "<username>":
      | permission   | granted   |
      | <perm1>      | true      |
      | <perm2>      | <granted> |
    Then user "<username>" should have <count> permissions

    Examples:
      | username | role   | perm1  | perm2  | granted | count |
      | dave     | admin  | read   | write  | true    | 2     |
      | eve      | viewer | read   | delete | false   | 2     |

  # Outline with no placeholders in step text (edge case)
  Scenario Outline: Static step text with varying data
    Given the application is running
    When I check the status
    Then the status code should be <code>

    Examples:
      | code |
      | 200  |
      | 404  |

  # Outline inside a Rule
  Rule: Access control

    Background:
      Given the access control module is loaded

    Scenario Outline: Permission check
      Given user "<user>" has role "<role>"
      When user "<user>" accesses "<resource>"
      Then access should be "<decision>"

      Examples: Admin access
        | user  | role  | resource   | decision |
        | frank | admin | dashboard  | granted  |
        | frank | admin | settings   | granted  |

      Examples: Viewer access
        | user  | role   | resource   | decision |
        | grace | viewer | dashboard  | granted  |
        | grace | viewer | settings   | denied   |
