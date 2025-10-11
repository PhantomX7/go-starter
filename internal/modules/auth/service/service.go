package service

import (
	"log"

	"github.com/PhantomX7/go-starter/internal/models"
	"github.com/PhantomX7/go-starter/internal/modules/user/repository"
	"github.com/PhantomX7/go-starter/pkg/config"
	"github.com/markbates/goth"
)

// AuthService defines the interface for auth service operations
type AuthService interface {
}

// authService implements the UserService interface
type authService struct {
	userRepository repository.UserRepository
}

// NewAuthService creates a new instance of AuthService
func NewAuthService(cfg *config.Config, userRepository repository.UserRepository) AuthService {
	
	return &authService{
		userRepository: userRepository,
	}
}

func (s *authService) CallbackOauth(user goth.User) error {

	log.Printf("CallbackOauth: %v", user)

    // // --- Repository Pattern Integration ---
    // dbUser, err := s.userRepository.FindById(user.UserID)
    // if err != nil {
    //     // User not found, create a new one
    //     newUser := &models.User{
    //         Provider:   gothUser.Provider,
    //         ProviderID: gothUser.UserID,
    //         Name:       gothUser.Name,
    //         Email:      gothUser.Email,
    //     }
    //     dbUser, err = userRepo.Create(newUser)
    //     if err != nil {
    //         c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
    //         return
    //     }
    // }

    // // --- NEW: Generate JWT ---
    // token, err := generateJWT(dbUser)
    // if err != nil {
    //     c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
    //     return
    // }

    // // --- NEW: Redirect to Frontend ---
    // // Redirect to a specific route on your frontend with the token
    // redirectURL := fmt.Sprintf("%s/auth/callback?token=%s", FRONTEND_URL, token)
    // c.Redirect(http.StatusTemporaryRedirect, redirectURL)

	return nil
}

// generateJWT creates a new JWT for a given user.
func generateJWT(user *models.User) (string, error) {
    // // Create the claims
    // claims := jwt.MapClaims{
    //     "id":    user.ID,
    //     "email": user.Email,
    //     "name":  user.Name,
    //     "exp":   time.Now().Add(time.Hour * 72).Unix(), // Token expires in 3 days
    //     "iat":   time.Now().Unix(),
    // }

    // // Create a new token object, specifying signing method and the claims
    // token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

    // // Sign and get the complete encoded token as a string using the secret
    // return token.SignedString(JWT_SECRET)
	return "", nil
}