#include <stdio.h>
#include <stdlib.h>

extern int Add(int a, int b);
extern int Multiply(int a, int b);
extern char* HelloWorld();
extern char* Base64Decode(char* encodedStr);

#ifdef DEBUG
extern void FreeString(char* str);
extern void PrintDebug(char* message);
#endif

int main() {
    printf("Testing Go shared library functions:\n");
    
    int sum = Add(5, 7);
    printf("Add(5, 7) = %d\n", sum);
    
    int product = Multiply(5, 7);
    printf("Multiply(5, 7) = %d\n", product);
    
    char* message = HelloWorld();
    printf("Message: %s\n", message);
    
    char* encoded = "SGVsbG8gV29ybGQh";  // "Hello World!" in base64
    printf("Encoded string: %s\n", encoded);
    
    char* decoded = Base64Decode(encoded);
    printf("Decoded string: %s\n", decoded);
    
    #ifdef DEBUG
    PrintDebug("Debug message from C");
    // Free the memory allocated by Go
    FreeString(message);
    printf("Free mem for $message \n");
    FreeString(decoded);
    printf("Free mem for $decoded \n");
    #endif
    
    return 0;
}