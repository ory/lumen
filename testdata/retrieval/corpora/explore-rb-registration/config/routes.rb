Rails.application.routes.draw do
  resources :users, only: %i[index create]
  get "/docs", to: "docs#show"
end
