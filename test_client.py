#!/usr/bin/env python3
"""
Simple gRPC client to test the AI Extension service
"""

import sys
import os
sys.path.append(os.path.join(os.path.dirname(__file__), 'src'))

import grpc
from ai_extension_pb2 import GenerateSQLRequest, DatabaseType
from ai_extension_pb2_grpc import AIExtensionStub

def test_sql_generation():
    """Test the SQL generation service"""
    # Create a gRPC channel
    channel = grpc.insecure_channel('localhost:50051')
    
    # Create a stub (client)
    stub = AIExtensionStub(channel)
    
    # Create a request
    request = GenerateSQLRequest(
        natural_language_input="Show me all users",
        database_type=DatabaseType.MYSQL
    )
    
    try:
        # Make the call
        print("Sending request: 'Show me all users' for MySQL database")
        response = stub.GenerateSQLFromNaturalLanguage(request)
        
        # Print the response
        if response.HasField('success'):
            print("\n✅ Success!")
            print(f"Generated SQL: {response.success.sql_query}")
            if response.success.HasField('explanation'):
                print(f"Explanation: {response.success.explanation}")
            if response.success.HasField('confidence_score'):
                print(f"Confidence Score: {response.success.confidence_score}")
        elif response.HasField('error'):
            print("\n❌ Error!")
            print(f"Error Code: {response.error.code}")
            print(f"Error Message: {response.error.message}")
        else:
            print("\n⚠️ Unknown response format")
            
    except grpc.RpcError as e:
        print(f"\n❌ gRPC Error: {e.code()} - {e.details()}")
    except Exception as e:
        print(f"\n❌ Unexpected Error: {str(e)}")
    finally:
        channel.close()

def test_with_schema():
    """Test SQL generation with schema information"""
    # Create a gRPC channel
    channel = grpc.insecure_channel('localhost:50051')
    
    # Create a stub (client)
    stub = AIExtensionStub(channel)
    
    # Create a request with schema
    schemas = [
        "CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(100), email VARCHAR(100), created_at TIMESTAMP);",
        "CREATE TABLE orders (id INT PRIMARY KEY, user_id INT, product_name VARCHAR(100), amount DECIMAL(10,2), order_date DATE);"
    ]
    
    request = GenerateSQLRequest(
        natural_language_input="Find all users who have placed orders in the last 30 days",
        database_type=DatabaseType.POSTGRESQL,
        schemas=schemas
    )
    
    try:
        # Make the call
        print("\nSending request: 'Find all users who have placed orders in the last 30 days' for PostgreSQL database with schema")
        response = stub.GenerateSQLFromNaturalLanguage(request)
        
        # Print the response
        if response.HasField('success'):
            print("\n✅ Success!")
            print(f"Generated SQL: {response.success.sql_query}")
            if response.success.HasField('explanation'):
                print(f"Explanation: {response.success.explanation}")
            if response.success.HasField('confidence_score'):
                print(f"Confidence Score: {response.success.confidence_score}")
        elif response.HasField('error'):
            print("\n❌ Error!")
            print(f"Error Code: {response.error.code}")
            print(f"Error Message: {response.error.message}")
        else:
            print("\n⚠️ Unknown response format")
            
    except grpc.RpcError as e:
        print(f"\n❌ gRPC Error: {e.code()} - {e.details()}")
    except Exception as e:
        print(f"\n❌ Unexpected Error: {str(e)}")
    finally:
        channel.close()

if __name__ == "__main__":
    print("🚀 Testing AI Extension gRPC Service")
    print("=" * 50)
    
    # Test basic SQL generation
    test_sql_generation()
    
    # Test with schema information
    test_with_schema()
    
    print("\n" + "=" * 50)
    print("✅ Testing completed!")