require "bcrypt"

class RegistrationService
  Result = Struct.new(:user, :errors) do
    def ok?
      errors.empty?
    end
  end

  def self.create(attrs)
    email = attrs[:email].to_s.strip.downcase
    return Result.new(nil, ["email is invalid"]) unless email.match?(URI::MailTo::EMAIL_REGEXP)
    return Result.new(nil, ["password is too short"]) if attrs[:password].to_s.length < 12

    digest = BCrypt::Password.create(attrs[:password])
    user = UserStore.new.insert(email: email, password_digest: digest, display_name: attrs[:display_name])
    WelcomeMailer.deliver_later(user.id)
    Result.new(user, [])
  rescue UserStore::DuplicateEmail
    Result.new(nil, ["email has already been taken"])
  end
end
