import unittest
from unittest.mock import patch, MagicMock
import sys
import os

# Add src to path for imports
sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), '..')))

from src.db.dialect_manager import query

class TestSQLAdapter(unittest.TestCase):
    """Tests for SQL adapter"""
    
    @patch('grpc.insecure_channel')
    def test_query(self, mock_channel):
        """Test query function"""
        # Setup
        mock_client = MagicMock()
        mock_response = MagicMock()
        mock_response.data = "test_data"
        mock_response.meta.tables = ["table1", "table2"]
        
        mock_client.Query.return_value = mock_response
        
        # Mock the LoaderStub import and instantiation
        with patch.dict('sys.modules', {
            'protos.ai_extension_pb2_grpc': MagicMock(),
            'protos.ai_extension_pb2': MagicMock()
        }):
            sys.modules['protos.ai_extension_pb2_grpc'].LoaderStub.return_value = mock_client
            
            # Call
            store = {
                "kind": {
                    "url": "test_url"
                }
            }
            data, meta = query(store, "SELECT * FROM users")
            
            # Assert
            self.assertEqual(data, "test_data")
            self.assertEqual(meta, {"tables": ["table1", "table2"]})

if __name__ == '__main__':
    unittest.main()