class UsersController < ApplicationController
  # GET /users
  def index
    @users = UserStore.new.list(limit: params.fetch(:limit, 50))
    render json: @users
  end

  # POST /users
  def create
    result = RegistrationService.create(user_params)
    if result.ok?
      render json: result.user, status: :created
    else
      render json: { errors: result.errors }, status: :unprocessable_entity
    end
  end

  private

  def user_params
    params.require(:user).permit(:email, :password, :display_name)
  end
end
