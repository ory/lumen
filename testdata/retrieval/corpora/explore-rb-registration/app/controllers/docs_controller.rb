class DocsController < ApplicationController
  # GET /docs
  def show
    render file: Rails.root.join("docs", "api.md"), content_type: "text/markdown"
  end
end
