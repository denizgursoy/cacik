Feature: IP address parameter type

  Scenario: Using IP type
    Given the server is at 192.168.1.1
    And the server is at ::1
    And the server is at 2001:db8::1
