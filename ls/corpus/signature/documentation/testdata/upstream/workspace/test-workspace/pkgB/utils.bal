# A utility function that processes data
#
# + data - The input data string
# + count - The number of times to repeat
# + return - The processed result string
public function processData(string data, int count) returns string {
    return data.repeat(count);
}

# A calculator function with multiple parameters
#
# + x - First operand
# + y - Second operand
# + operation - The operation to perform
# + return - The calculation result
public function calculate(float x, float y, string operation) returns float {
    if operation == "add" {
        return x + y;
    } else if operation == "multiply" {
        return x * y;
    }
    return 0.0;
}

# A public class with methods
public class DataProcessor {
    private string name;

    # Initialize the processor
    #
    # + name - The processor name
    public function init(string name) {
        self.name = name;
    }

    # Process a value with options
    #
    # + value - The value to process
    # + uppercase - Whether to convert to uppercase
    # + return - The processed value
    public function process(string value, boolean uppercase) returns string {
        if uppercase {
            return value.toUpperAscii();
        }
        return value;
    }

    # Transform data with multiple parameters
    #
    # + input - Input string
    # + prefix - Prefix to add
    # + suffix - Suffix to add
    # + return - Transformed string
    public function transform(string input, string prefix, string suffix) returns string {
        return prefix + input + suffix;
    }
}

# A client with actions
public client class ApiClient {
    private string endpoint;

    # Initialize the API client
    #
    # + endpoint - The API endpoint URL
    public function init(string endpoint) {
        self.endpoint = endpoint;
    }

    # Fetch data from the API
    #
    # + id - The resource ID
    # + includeMetadata - Whether to include metadata
    # + return - The fetched data or an error
    remote function fetch(int id, boolean includeMetadata) returns string|error {
        return "data-" + id.toString();
    }

    # Send data to the API
    #
    # + payload - The data payload
    # + headers - Custom headers
    # + timeout - Request timeout in seconds
    # + return - Success status or an error
    remote function send(string payload, map<string> headers, int timeout) returns boolean|error {
        return true;
    }
}

# Represents a person with basic information
public type Person record {
    # the person's name
    string name;
    # the person's age
    int age;
    # the person's email address
    string email;
};
