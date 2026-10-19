Tóm tắt Idea của gocore
gocore là một Backend Foundation Toolkit (Microservice Chassis) viết bằng Go — đóng vai trò làm "bộ khung" hạ tầng dùng chung cho tất cả các service hiện tại và tương lai.

3 Trụ cột cốt lõi:
Chuẩn hóa quy chuẩn (Standardization):

Thống nhất format mã lỗi (

stderr
), cấu trúc response API (

stdres
), cơ chế log (

stdlog
) và giám sát (Prometheus, Sentry, Graylog).
Thống nhất cơ chế phân quyền, xác thực (IAM, Casbin, OAuth, Firebase).
Đóng gói hạ tầng (Infrastructure Wrappers):

Đóng gói sẵn các kết nối phổ biến nhất để dùng ngay với cấu hình chuẩn: Database (Postgres/MySQL), Cache (Redis), Queue (Kafka), Network (HTTP3/Gin, ConnectRPC, WebSocket).
Tích hợp sẵn cơ chế chống lỗi: connection pool, retry, distributed lock, timeout.
Tối ưu tốc độ phát triển (Plug & Play):

Loại bỏ code lặp lại (boilerplate) khi tạo service mới.
Dựng một microservice mới chỉ mất vài phút: chỉ cần import gocore và tập trung 100% vào logic nghiệp vụ.
Áp dụng các best practice hiện đại trong hệ sinh thái Go.

Mục tiêu ngắn hạn trước tiên là tạo để apply thay thế cho những gì có trong project:
/home/ubuntu/my_projects/kdc/kdc_bike_BE trước giúp project này gọn gàng ngăn nắp hơn
