#External Mocks
mockgen github.com/diskfs/go-diskfs/filesystem FileSystem > services/filesystem_mock_test.go
sed -i 's/package mock_filesystem/package services/g' services/filesystem_mock_test.go

#Internal Mocks

mockgen -source=clients/osClient.go -destination=clients/osClient_mock.go
sed -i 's/package mock_clients/package clients/g' clients/osClient_mock.go
sed -i 's/clients\.//g' clients/osClient_mock.go
sed -i 's~clients "zs-vm-agent/clients"~~g' clients/osClient_mock.go

mockgen -source=clients/fileWrapper.go -destination=clients/fileWrapper_mock.go
sed -i 's/package mock_clients/package clients/g' clients/fileWrapper_mock.go
sed -i 's/clients\.//g' clients/fileWrapper_mock.go
sed -i 's~clients "zs-vm-agent/clients"~~g' clients/fileWrapper_mock.go

mockgen -source=clients/filesystemWrapper.go -destination=clients/filesystemWrapper_mock.go
sed -i 's/package mock_clients/package clients/g' clients/filesystemWrapper_mock.go
sed -i 's/clients\.//g' clients/filesystemWrapper_mock.go
sed -i 's~clients "zs-vm-agent/clients"~~g' clients/filesystemWrapper_mock.go

mockgen -source=clients/diskWrapper.go -destination=clients/diskWrapper_mock.go
sed -i 's/package mock_clients/package clients/g' clients/diskWrapper_mock.go
sed -i 's/clients\.//g' clients/diskWrapper_mock.go
sed -i 's~clients "zs-vm-agent/clients"~~g' clients/diskWrapper_mock.go

mockgen -source=clients/userClient.go -destination=clients/userClient_mock.go
sed -i 's/package mock_clients/package clients/g' clients/userClient_mock.go
sed -i 's/clients\.//g' clients/userClient_mock.go
sed -i 's~clients "zs-vm-agent/clients"~~g' clients/userClient_mock.go