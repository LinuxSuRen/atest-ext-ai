import unittest
from unittest.mock import patch, MagicMock
import sys
import os

# Add src to path for imports
sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), '..')))

from src.grpc_server import AIExtensionServicer

class TestGRPCServer(unittest.TestCase):
    """Tests for gRPC server"""
    
    def setUp(self):
        """Set up test fixtures"""
        self.config = MagicMock()
        self.servicer = AIExtensionServicer(self.config)
    
    @patch('src.grpc_server.query')
    def test_query(self, mock_query):
        """Test Query method"""
        # Setup
        mock_query.return_value = ("test_data", {"tables": ["table1", "table2"]})
        
        request = MagicMock()
        request.sql = "SELECT * FROM users"
        
        context = MagicMock()
        
        # Call
        response = self.servicer.Query(request, context)
        
        # Assert
        self.assertEqual(response.data, "test_data")
        self.assertEqual(list(response.meta.tables), ["table1", "table2"])

if __name__ == '__main__':
    unittest.main()