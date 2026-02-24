@billing
Feature: Verify billing

  @important
  Scenario: Missing product description
    Given hello

  Scenario: Several products
    Given hello


  Scenario Outline: Steps will run conditionally if tagged
    Given user is logged in
    When user clicks <link>
    Then user will be logged out

    @mobile
    Examples:
      | link                  |
      | logout link on mobile |

    @desktop
    Examples:
      | link                   |
      | logout link on desktop |