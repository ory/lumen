class UserStore
  class DuplicateEmail < StandardError; end

  User = Struct.new(:id, :email, :display_name, keyword_init: true)

  def initialize(db = DB.connection)
    @db = db
  end

  def insert(email:, password_digest:, display_name:)
    row = @db.exec_params(
      "INSERT INTO users (email, password_digest, display_name) VALUES ($1, $2, $3) RETURNING id",
      [email, password_digest, display_name]
    ).first
    User.new(id: row["id"].to_i, email: email, display_name: display_name)
  rescue PG::UniqueViolation
    raise DuplicateEmail
  end

  def list(limit:)
    @db.exec_params("SELECT id, email, display_name FROM users ORDER BY id LIMIT $1", [limit]).map do |row|
      User.new(id: row["id"].to_i, email: row["email"], display_name: row["display_name"])
    end
  end
end
